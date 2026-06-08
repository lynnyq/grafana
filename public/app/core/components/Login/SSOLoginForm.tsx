import { css } from '@emotion/css';
import { useId } from 'react';
import { useForm } from 'react-hook-form';

import { type GrafanaTheme2 } from '@grafana/data';
import { t } from '@grafana/i18n';
import { Button, Input, Field, useStyles2 } from '@grafana/ui';

import { PasswordField } from '../PasswordField/PasswordField';

export interface SSOFormModel {
  user: string;
  password: string;
}

interface Props {
  onSubmit: (data: SSOFormModel) => void;
  isLoggingIn: boolean;
}

export const SSOLoginForm = ({ onSubmit, isLoggingIn }: Props) => {
  const styles = useStyles2(getStyles);
  const usernameId = useId();
  const passwordId = useId();
  const {
    handleSubmit,
    register,
    formState: { errors },
  } = useForm<SSOFormModel>({ mode: 'onChange' });

  return (
    <div className={styles.wrapper}>
      <form onSubmit={handleSubmit(onSubmit)}>
        <Field
          label={t('login.sso.form.username-label', 'Email or username')}
          invalid={!!errors.user}
          error={errors.user?.message}
          noMargin
        >
          <Input
            {...register('user', { required: t('login.sso.form.username-required', 'Email or username is required') })}
            id={usernameId}
            autoFocus
            autoCapitalize="none"
            placeholder={t('login.sso.form.username-placeholder', 'email or username')}
          />
        </Field>
        <Field
          label={t('login.sso.form.password-label', 'Password')}
          invalid={!!errors.password}
          error={errors.password?.message}
          noMargin
        >
          <PasswordField
            {...register('password', { required: t('login.sso.form.password-required', 'Password is required') })}
            id={passwordId}
            autoComplete="current-password"
            placeholder={t('login.sso.form.password-placeholder', 'password')}
          />
        </Field>
        <Button type="submit" className={styles.submitButton} disabled={isLoggingIn}>
          {isLoggingIn ? t('login.sso.form.submit-loading-label', 'Logging in...') : t('login.sso.form.submit-label', 'Log in')}
        </Button>
      </form>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => {
  return {
    wrapper: css({
      width: '100%',
      paddingBottom: theme.spacing(2),
    }),
    submitButton: css({
      justifyContent: 'center',
      width: '100%',
    }),
  };
};
