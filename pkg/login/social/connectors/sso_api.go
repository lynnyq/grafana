package connectors

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/login/social"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/setting"
)

var _ social.SocialConnector = (*SocialSSO)(nil)

type SocialSSO struct {
	info        *social.OAuthInfo
	cfg         *setting.Cfg
	log         log.Logger
	ssoApiUrl   string
	rsaPubKey   string
	defaultRole string
	httpClient  *http.Client
}

func NewSSOProvider(info *social.OAuthInfo, cfg *setting.Cfg) *SocialSSO {
	s := &SocialSSO{
		info:        info,
		cfg:         cfg,
		log:         log.New("oauth.sso"),
		ssoApiUrl:   cfg.SSOAuth.SsoApiUrl,
		rsaPubKey:   cfg.SSOAuth.RSAPublicKey,
		defaultRole: cfg.SSOAuth.DefaultRole,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.SSOAuth.TlsSkipVerify,
				},
			},
		},
	}

	return s
}

// ssoRequest is the request body sent to the SSO API
type ssoRequest struct {
	LoginName string `json:"loginName"`
	Password  string `json:"password"`
}

// ssoResponse is the response from the SSO API
type ssoResponse struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Content *ssoUser `json:"content"`
}

// ssoUser represents user info from the SSO API response
type ssoUser struct {
	ID          int64  `json:"id"`
	LoginName   string `json:"loginName"`
	UserName    string `json:"userName"`
	IDCardName  string `json:"idCardName"`
	MobileNo    string `json:"mobileNo"`
	Email       string `json:"email"`
	DeptNo      string `json:"deptNo"`
	WorkID      string `json:"workId"`
	Gender      string `json:"gender"`
	OfficePhone string `json:"officePhone"`
}

// Authenticate calls the SSO API with username and encrypted password to verify the user
func (s *SocialSSO) Authenticate(ctx context.Context, username, password string) (*social.BasicUserInfo, error) {
	encryptedPassword, err := s.encryptPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	ssoReq := ssoRequest{
		LoginName: username,
		Password:  encryptedPassword,
	}

	reqBody, err := json.Marshal(ssoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SSO request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ssoApiUrl, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create SSO request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call SSO service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.log.Warn("Failed to close SSO response body", "err", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSO response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SSO service returned status %d: %s", resp.StatusCode, string(body))
	}

	var ssoResp ssoResponse
	if err := json.Unmarshal(body, &ssoResp); err != nil {
		return nil, fmt.Errorf("failed to decode SSO response: %w", err)
	}

	if ssoResp.Code != "3000" || ssoResp.Content == nil {
		errMsg := ssoResp.Message
		if errMsg == "" {
			errMsg = "SSO authentication failed"
		}
		return nil, fmt.Errorf("SSO authentication failed: %s", errMsg)
	}

	ssoUser := ssoResp.Content
	loginName := ssoUser.LoginName
	if loginName == "" {
		loginName = username
	}

	displayName := ssoUser.IDCardName
	if displayName == "" {
		displayName = ssoUser.UserName
	}

	userInfo := &social.BasicUserInfo{
		Id:    fmt.Sprintf("%d", ssoUser.ID),
		Name:  displayName,
		Email: ssoUser.Email,
		Login: loginName,
	}

	role := org.RoleType(s.defaultRole)
	if role.IsValid() {
		userInfo.Role = role
	} else {
		userInfo.Role = org.RoleViewer
	}

	s.log.Debug("SSO authentication successful", "username", loginName, "email", ssoUser.Email)
	return userInfo, nil
}

// encryptPassword encrypts the password using RSA public key (PKCS#1 v1.5)
func (s *SocialSSO) encryptPassword(password string) (string, error) {
	publicKey := strings.TrimSpace(s.rsaPubKey)
	if publicKey == "" {
		return "", fmt.Errorf("SSO RSA public key is not configured")
	}

	var keyBytes []byte
	var err error

	keyBytes, err = base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		block, _ := pem.Decode([]byte(publicKey))
		if block == nil {
			return "", fmt.Errorf("failed to parse public key: neither base64 nor PEM format")
		}
		keyBytes = block.Bytes
	}

	var rsaPub *rsa.PublicKey
	rsaPub, err = x509.ParsePKCS1PublicKey(keyBytes)
	if err != nil {
		pub, parseErr := x509.ParsePKIXPublicKey(keyBytes)
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse RSA public key: %w", parseErr)
		}
		var ok bool
		rsaPub, ok = pub.(*rsa.PublicKey)
		if !ok {
			return "", fmt.Errorf("not an RSA public key")
		}
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(password))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt password: %w", err)
	}

	return hex.EncodeToString(encrypted), nil
}

// SocialConnector interface implementation

func (s *SocialSSO) UserInfo(ctx context.Context, client *http.Client, token *oauth2.Token) (*social.BasicUserInfo, error) {
	return nil, fmt.Errorf("SSO provider does not support OAuth2 UserInfo flow, use Authenticate instead")
}

func (s *SocialSSO) IsEmailAllowed(email string) bool {
	return true
}

func (s *SocialSSO) IsSignupAllowed() bool {
	return s.info.AllowSignup
}

func (s *SocialSSO) GetOAuthInfo() *social.OAuthInfo {
	return s.info
}

func (s *SocialSSO) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return ""
}

func (s *SocialSSO) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return nil, nil
}

func (s *SocialSSO) Client(ctx context.Context, t *oauth2.Token) *http.Client {
	return s.httpClient
}

func (s *SocialSSO) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	return nil
}

func (s *SocialSSO) SupportBundleContent(bf *bytes.Buffer) error {
	bf.WriteString("## SSO specific configuration\n\n")
	bf.WriteString("```ini\n")
	fmt.Fprintf(bf, "sso_api_url = %s\n", s.ssoApiUrl)
	fmt.Fprintf(bf, "default_role = %s\n", s.defaultRole)
	fmt.Fprintf(bf, "tls_skip_verify_insecure = %v\n", s.cfg.SSOAuth.TlsSkipVerify)
	bf.WriteString("```\n\n")
	return nil
}
