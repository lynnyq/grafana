// Libraries
import { css } from '@emotion/css';
import { useState, useCallback } from 'react';

// Components
import { type GrafanaTheme2, PageLayoutType } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { config, type FetchError, getBackendSrv, isFetchError } from '@grafana/runtime';
import { Alert, LinkButton, Stack, Tab, TabsBar, TabContent, useStyles2 } from '@grafana/ui';
import { Branding } from 'app/core/components/Branding/Branding';

import { ChangePassword } from '../ForgottenPassword/ChangePassword';
import { Page } from '../Page/Page';

import LoginCtrl from './LoginCtrl';
import { LoginForm } from './LoginForm';
import { LoginLayout, InnerBox } from './LoginLayout';
import { LoginServiceButtons } from './LoginServiceButtons';
import { SSOLoginForm, type SSOFormModel } from './SSOLoginForm';
import { UserSignup } from './UserSignup';
import { type LoginDTO } from './types';

const LoginPage = () => {
  const styles = useStyles2(getStyles);
  const [ssoActiveTab, setSsoActiveTab] = useState<'sso' | 'local'>('sso');
  const [isSsoLoggingIn, setIsSsoLoggingIn] = useState(false);
  const [ssoLoginErrorMessage, setSsoLoginErrorMessage] = useState<string | undefined>();

  document.title = Branding.AppTitle;

  const ssoEnabled = config.ssoEnabled;

  const handleSsoSubmit = useCallback(
    async (formModel: SSOFormModel) => {
      setSsoLoginErrorMessage(undefined);
      setIsSsoLoggingIn(true);
      try {
        const result = await getBackendSrv().post<LoginDTO>('/sso/login', formModel, { showErrorAlert: false });
        setIsSsoLoggingIn(false);
        if (result?.redirectUrl) {
          if (config.appSubUrl !== '' && !result.redirectUrl.startsWith(config.appSubUrl)) {
            window.location.assign(config.appSubUrl + result.redirectUrl);
          } else {
            window.location.assign(result.redirectUrl);
          }
        } else {
          window.location.assign(config.appSubUrl + '/');
        }
      } catch (err) {
        setIsSsoLoggingIn(false);
        const fetchErrorMessage = isFetchError(err) ? getSsoErrorMessage(err) : undefined;
        setSsoLoginErrorMessage(fetchErrorMessage || t('login.sso.error.unknown', 'SSO login failed'));
      }
    },
    []
  );

  return (
    <Page layout={PageLayoutType.Custom}>
      <LoginCtrl>
        {({
          loginHint,
          passwordHint,
          disableLoginForm,
          disableUserSignUp,
          login,
          isLoggingIn,
          changePassword,
          skipPasswordChange,
          isChangingPassword,
          showDefaultPasswordWarning,
          loginErrorMessage,
        }) => (
          <LoginLayout isChangingPassword={isChangingPassword}>
            {!isChangingPassword && (
              <InnerBox>
                {(ssoEnabled ? ssoActiveTab === 'sso' : true) && (loginErrorMessage || ssoLoginErrorMessage) && (
                  <Alert className={styles.alert} severity="error" title={t('login.error.title', 'Login failed')}>
                    {ssoEnabled && ssoActiveTab === 'sso' ? ssoLoginErrorMessage : loginErrorMessage}
                  </Alert>
                )}

                {ssoEnabled && (
                  <>
                    <TabsBar>
                      <Tab
                        label={config.ssoName || t('login.sso.tab-label', 'SSO Login')}
                        active={ssoActiveTab === 'sso'}
                        onChangeTab={() => {
                          setSsoActiveTab('sso');
                          setSsoLoginErrorMessage(undefined);
                        }}
                      />
                      <Tab
                        label={t('login.local.tab-label', 'Local Login')}
                        active={ssoActiveTab === 'local'}
                        onChangeTab={() => setSsoActiveTab('local')}
                      />
                    </TabsBar>
                    <TabContent>
                      {ssoActiveTab === 'sso' && (
                        <SSOLoginForm onSubmit={handleSsoSubmit} isLoggingIn={isSsoLoggingIn} />
                      )}
                      {ssoActiveTab === 'local' && (
                        <>
                          {!disableLoginForm && (
                            <LoginForm
                              onSubmit={login}
                              loginHint={loginHint}
                              passwordHint={passwordHint}
                              isLoggingIn={isLoggingIn}
                            >
                              <Stack justifyContent="flex-end">
                                {!config.auth.disableLogin && (
                                  <LinkButton
                                    className={styles.forgottenPassword}
                                    fill="text"
                                    href={`${config.appSubUrl}/user/password/send-reset-email`}
                                  >
                                    <Trans i18nKey="login.forgot-password">Forgot your password?</Trans>
                                  </LinkButton>
                                )}
                              </Stack>
                            </LoginForm>
                          )}
                          <LoginServiceButtons />
                          {!disableUserSignUp && <UserSignup />}
                        </>
                      )}
                    </TabContent>
                  </>
                )}

                {!ssoEnabled && (
                  <>
                    {!disableLoginForm && (
                      <LoginForm
                        onSubmit={login}
                        loginHint={loginHint}
                        passwordHint={passwordHint}
                        isLoggingIn={isLoggingIn}
                      >
                        <Stack justifyContent="flex-end">
                          {!config.auth.disableLogin && (
                            <LinkButton
                              className={styles.forgottenPassword}
                              fill="text"
                              href={`${config.appSubUrl}/user/password/send-reset-email`}
                            >
                              <Trans i18nKey="login.forgot-password">Forgot your password?</Trans>
                            </LinkButton>
                          )}
                        </Stack>
                      </LoginForm>
                    )}
                    <LoginServiceButtons />
                    {!disableUserSignUp && <UserSignup />}
                  </>
                )}
              </InnerBox>
            )}

            {isChangingPassword && (
              <InnerBox>
                <ChangePassword
                  showDefaultPasswordWarning={showDefaultPasswordWarning}
                  onSubmit={changePassword}
                  onSkip={() => skipPasswordChange()}
                />
              </InnerBox>
            )}
          </LoginLayout>
        )}
      </LoginCtrl>
    </Page>
  );
};

export default LoginPage;

function getSsoErrorMessage(err: FetchError<undefined | { messageId?: string; message?: string }>): string | undefined {
  switch (err.data?.messageId) {
    case 'password-auth.empty':
    case 'password-auth.failed':
    case 'password-auth.invalid':
      return t('login.sso.error.invalid-user-or-password', 'Invalid username or password');
    case 'login-attempt.blocked':
      return t(
        'login.sso.error.blocked',
        'You have exceeded the number of login attempts for this user. Please try again later.'
      );
    default:
      return err.data?.message;
  }
}

const getStyles = (theme: GrafanaTheme2) => {
  return {
    forgottenPassword: css({
      padding: 0,
      marginTop: theme.spacing(0.5),
    }),

    alert: css({
      width: '100%',
    }),
  };
};
