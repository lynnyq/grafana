package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/metrics"
	"github.com/grafana/grafana/pkg/infra/network"
	"github.com/grafana/grafana/pkg/login/social"
	"github.com/grafana/grafana/pkg/login/social/connectors"
	loginservice "github.com/grafana/grafana/pkg/services/login"
	"github.com/grafana/grafana/pkg/services/auth"
	"github.com/grafana/grafana/pkg/services/authn"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/web"
)

var ssoLog = log.New("api.sso-login")

type ssoLoginRequest struct {
	User     string `json:"user" binding:"Required"`
	Password string `json:"password" binding:"Required"`
}

func (hs *HTTPServer) SSOLoginPost(c *contextmodel.ReqContext) response.Response {
	if !hs.Cfg.SSOAuth.Enabled {
		return response.Error(http.StatusBadRequest, "SSO login is not enabled", nil)
	}

	loginReq := ssoLoginRequest{}
	if err := web.Bind(c.Req, &loginReq); err != nil {
		return response.Error(http.StatusBadRequest, "Invalid request", err)
	}

	if loginReq.User == "" || loginReq.Password == "" {
		return response.Error(http.StatusBadRequest, "Username and password are required", nil)
	}

	// Get the SSO connector
	conn, err := hs.SocialService.GetConnector(social.SSOProviderName)
	if err != nil {
		ssoLog.Error("Failed to get SSO connector", "error", err)
		return response.Error(http.StatusInternalServerError, "SSO provider not configured", err)
	}

	ssoConnector, ok := conn.(*connectors.SocialSSO)
	if !ok {
		ssoLog.Error("SSO connector is not of expected type")
		return response.Error(http.StatusInternalServerError, "SSO provider misconfigured", nil)
	}

	// Authenticate via SSO API
	userInfo, err := ssoConnector.Authenticate(c.Req.Context(), loginReq.User, loginReq.Password)
	if err != nil {
		ssoLog.Info("SSO authentication failed", "username", loginReq.User, "error", err)
		return response.Error(http.StatusUnauthorized, "Invalid username or password", err)
	}

	// Check if signup is allowed for new users
	if !ssoConnector.IsSignupAllowed() {
		existingUser, lookupErr := hs.userService.GetByLogin(c.Req.Context(), &user.GetUserByLoginQuery{LoginOrEmail: userInfo.Login})
		if lookupErr != nil || existingUser == nil {
			ssoLog.Info("SSO login rejected: user does not exist and signup is disabled", "login", userInfo.Login)
			return response.Error(http.StatusForbidden, "Sign up is not allowed for this SSO provider", nil)
		}
	}

	// Find or create the Grafana user
	usr, err := hs.getOrCreateSSOUser(c.Req.Context(), userInfo)
	if err != nil {
		ssoLog.Error("Failed to get or create SSO user", "login", userInfo.Login, "error", err)
		return response.Error(http.StatusInternalServerError, "Failed to process user", err)
	}

	// Create auth token and session
	addr := c.RemoteAddr()
	ip, ipErr := network.GetIPFromAddress(addr)
	if ipErr != nil {
		ssoLog.Debug("Failed to get IP from client address", "addr", addr)
		ip = nil
	}

	ctx := context.WithValue(c.Req.Context(), loginservice.RequestURIKey{}, c.Req.RequestURI)
	userToken, err := hs.AuthTokenService.CreateToken(ctx, &auth.CreateTokenCommand{
		User:      &user.User{ID: usr.ID, Email: usr.Email, Login: usr.Login},
		ClientIP:  ip,
		UserAgent: c.Req.UserAgent(),
	})
	if err != nil {
		ssoLog.Error("Failed to create auth token for SSO user", "login", userInfo.Login, "error", err)
		return response.Error(http.StatusInternalServerError, "Failed to create session", err)
	}

	// Set auth info for the SSO module
	setAuthInfoCmd := &loginservice.SetAuthInfoCommand{
		AuthModule: loginservice.SSOAuthModule,
		AuthId:     userInfo.Id,
		UserId:     usr.ID,
		UserUID:    usr.UID,
		OAuthToken: nil,
	}
	if err := hs.authInfoService.SetAuthInfo(c.Req.Context(), setAuthInfoCmd); err != nil {
		ssoLog.Error("Failed to set auth info for SSO user", "login", userInfo.Login, "error", err)
	}

	// Write session cookie
	authn.WriteSessionCookie(c.Resp, hs.Cfg, userToken)

	metrics.MApiLoginPost.Inc()
	ssoLog.Info("Successful SSO login", "user", usr.Email)

	return response.JSON(http.StatusOK, map[string]any{
		"message":     "Logged in",
		"redirectUrl": hs.Cfg.AppSubURL + "/",
	})
}

// getOrCreateSSOUser finds an existing user or creates a new one based on SSO user info
func (hs *HTTPServer) getOrCreateSSOUser(ctx context.Context, userInfo *social.BasicUserInfo) (*user.User, error) {
	// Try to find user by login
	usr, err := hs.userService.GetByLogin(ctx, &user.GetUserByLoginQuery{LoginOrEmail: userInfo.Login})
	if err == nil && usr != nil {
		return usr, nil
	}

	// Try to find user by email if login lookup failed
	if userInfo.Email != "" {
		usr, err = hs.userService.GetByEmail(ctx, &user.GetUserByEmailQuery{Email: userInfo.Email})
		if err == nil && usr != nil {
			return usr, nil
		}
	}

	// Create new user
	createCmd := user.CreateUserCommand{
		Login:   userInfo.Login,
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		OrgID:   int64(hs.Cfg.AutoAssignOrgId),
		IsAdmin: false,
	}

	if hs.Cfg.AutoAssignOrg {
		createCmd.OrgID = int64(hs.Cfg.AutoAssignOrgId)
	}

	createdUser, err := hs.userService.Create(ctx, &createCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSO user: %w", err)
	}

	ssoLog.Info("Auto-created user from SSO login", "login", userInfo.Login, "email", userInfo.Email)
	return createdUser, nil
}
