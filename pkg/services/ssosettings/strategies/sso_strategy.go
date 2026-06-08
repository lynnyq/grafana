package strategies

import (
	"context"

	"github.com/grafana/grafana/pkg/login/social"
	"github.com/grafana/grafana/pkg/services/ssosettings"
	"github.com/grafana/grafana/pkg/setting"
)

type SSOStrategy struct {
	cfg     *setting.Cfg
	settings map[string]any
}

var _ ssosettings.FallbackStrategy = (*SSOStrategy)(nil)

func NewSSOStrategy(cfg *setting.Cfg) *SSOStrategy {
	s := &SSOStrategy{
		cfg:      cfg,
		settings: make(map[string]any),
	}
	s.loadSettings()
	return s
}

func (s *SSOStrategy) IsMatch(provider string) bool {
	return provider == social.SSOProviderName
}

func (s *SSOStrategy) GetProviderConfig(_ context.Context, _ string) (map[string]any, error) {
	result := make(map[string]any, len(s.settings))
	for k, v := range s.settings {
		result[k] = v
	}
	return result, nil
}

func (s *SSOStrategy) loadSettings() {
	section := s.cfg.Raw.Section("auth." + social.SSOProviderName)

	s.settings = map[string]any{
		"enabled":                  section.Key("enabled").MustBool(false),
		"name":                     section.Key("name").MustString("SSO"),
		"icon":                     section.Key("icon").MustString("signin"),
		"allow_sign_up":            section.Key("allow_sign_up").MustBool(true),
		"auto_login":               section.Key("auto_login").MustBool(false),
		"sso_api_url":              section.Key("sso_api_url").Value(),
		"sso_rsa_public_key":       section.Key("sso_rsa_public_key").Value(),
		"default_role":             section.Key("default_role").MustString("Viewer"),
		"tls_skip_verify_insecure": section.Key("tls_skip_verify_insecure").MustBool(false),
		"skip_org_role_sync":       true,
	}
}
