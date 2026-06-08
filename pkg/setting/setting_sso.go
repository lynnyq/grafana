package setting

type SSOAuthSettings struct {
	Enabled         bool
	Name            string
	Icon            string
	AllowSignUp     bool
	AutoLogin       bool
	SsoApiUrl       string
	RSAPublicKey    string
	DefaultRole     string
	TlsSkipVerify   bool
}

func (cfg *Cfg) readSSOAuthSettings() {
	ssoAuthSettings := SSOAuthSettings{}
	ssoSec := cfg.Raw.Section("auth.sso")
	ssoAuthSettings.Enabled = ssoSec.Key("enabled").MustBool(false)
	ssoAuthSettings.Name = valueAsString(ssoSec, "name", "SSO")
	ssoAuthSettings.Icon = valueAsString(ssoSec, "icon", "signin")
	ssoAuthSettings.AllowSignUp = ssoSec.Key("allow_sign_up").MustBool(true)
	ssoAuthSettings.AutoLogin = ssoSec.Key("auto_login").MustBool(false)
	ssoAuthSettings.SsoApiUrl = valueAsString(ssoSec, "sso_api_url", "")
	ssoAuthSettings.RSAPublicKey = valueAsString(ssoSec, "sso_rsa_public_key", "")
	ssoAuthSettings.DefaultRole = valueAsString(ssoSec, "default_role", "Viewer")
	ssoAuthSettings.TlsSkipVerify = ssoSec.Key("tls_skip_verify_insecure").MustBool(false)

	cfg.SSOAuth = ssoAuthSettings
}
