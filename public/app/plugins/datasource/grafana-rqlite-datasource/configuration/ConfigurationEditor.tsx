import {
  type DataSourcePluginOptionsEditorProps,
  onUpdateDatasourceJsonDataOption,
  onUpdateDatasourceSecureJsonDataOption,
  updateDatasourcePluginJsonDataOption,
  updateDatasourcePluginResetOption,
} from '@grafana/data';
import { ConfigSection, DataSourceDescription } from '@grafana/plugin-ui';
import { MaxLifetimeField, MaxOpenConnectionsField } from '@grafana/sql';
import {
  Combobox,
  type ComboboxOption,
  Divider,
  Field,
  Input,
  SecretInput,
  Stack,
  Switch,
  TextArea,
} from '@grafana/ui';

import { ConnectionMode, type RqliteOptions, ReadConsistency, type SecureJsonData } from '../types';

export const RqliteConfigEditor = (props: DataSourcePluginOptionsEditorProps<RqliteOptions, SecureJsonData>) => {
  const { options, onOptionsChange } = props;
  const jsonData = options.jsonData;

  const onResetPassword = () => {
    updateDatasourcePluginResetOption(props, 'password');
  };

  const connectionModeOptions: Array<ComboboxOption<ConnectionMode>> = [
    { value: ConnectionMode.AutoDiscovery, label: 'Auto-discovery' },
    { value: ConnectionMode.StaticList, label: 'Static list' },
  ];

  const readConsistencyOptions: Array<ComboboxOption<ReadConsistency>> = [
    { value: ReadConsistency.None, label: 'None' },
    { value: ReadConsistency.Weak, label: 'Weak' },
    { value: ReadConsistency.Strong, label: 'Strong' },
    { value: ReadConsistency.Linearizable, label: 'Linearizable' },
  ];

  const onConnectionModeSelected = (value: ComboboxOption<ConnectionMode>) => {
    updateDatasourcePluginJsonDataOption(props, 'connectionMode', value.value);
  };

  const onReadConsistencySelected = (value: ComboboxOption<ReadConsistency>) => {
    updateDatasourcePluginJsonDataOption(props, 'readConsistency', value.value);
  };

  const onMaxConnectionsChanged = (number?: number) => {
    updateDatasourcePluginJsonDataOption(props, 'maxOpenConns', number);
  };

  const onMaxLifetimeChanged = (number?: number) => {
    updateDatasourcePluginJsonDataOption(props, 'connMaxLifetime', number);
  };

  const onTLSSkipVerifyChanged = (event: React.ChangeEvent<HTMLInputElement>) => {
    updateDatasourcePluginJsonDataOption(props, 'tlsSkipVerify', event.currentTarget.checked);
  };

  const WIDTH_LONG = 40;

  return (
    <>
      <DataSourceDescription
        dataSourceName="rqlite"
        docsLink="https://rqlite.io/docs/"
        hasRequiredFields={true}
      />

      <Divider />

      <ConfigSection title="Connection">
        <Stack direction="column" gap={2}>
          <Field label="Connection Mode" description="How to connect to the rqlite cluster">
            <Combobox
              options={connectionModeOptions}
              value={jsonData.connectionMode || ConnectionMode.AutoDiscovery}
              onChange={onConnectionModeSelected}
              width={WIDTH_LONG}
            />
          </Field>

          {(jsonData.connectionMode === ConnectionMode.AutoDiscovery ||
            !jsonData.connectionMode) && (
            <Field label="Host URL" required description="Seed node URL for auto-discovery">
              <Input
                width={WIDTH_LONG}
                name="url"
                type="text"
                value={options.url || ''}
                placeholder="http://localhost:4001"
                onChange={(event) =>
                  onOptionsChange({ ...options, url: event.currentTarget.value })
                }
              />
            </Field>
          )}

          {jsonData.connectionMode === ConnectionMode.StaticList && (
            <Field
              label="Cluster URLs"
              required
              description="Comma-separated list of rqlite node URLs"
            >
              <TextArea
                value={jsonData.clusterUrls || ''}
                placeholder="http://node1:4001,http://node2:4001,http://node3:4001"
                onChange={onUpdateDatasourceJsonDataOption(props, 'clusterUrls')}
                rows={3}
              />
            </Field>
          )}

          <Field
            label="Read Consistency"
            description="Controls how up-to-date the data returned by read queries is"
          >
            <Combobox
              options={readConsistencyOptions}
              value={jsonData.readConsistency || ReadConsistency.Weak}
              onChange={onReadConsistencySelected}
              width={WIDTH_LONG}
            />
          </Field>
        </Stack>
      </ConfigSection>

      <Divider />

      <ConfigSection title="Authentication">
        <Stack direction="column" gap={2}>
          <Field label="Username" description="Basic auth username for rqlite">
            <Input
              width={WIDTH_LONG}
              value={jsonData.username || ''}
              placeholder="Username"
              onChange={onUpdateDatasourceJsonDataOption(props, 'username')}
            />
          </Field>

          <Field label="Password" description="Basic auth password for rqlite">
            <SecretInput
              width={WIDTH_LONG}
              placeholder="Password"
              isConfigured={options.secureJsonFields && options.secureJsonFields.password}
              onReset={onResetPassword}
              onBlur={onUpdateDatasourceSecureJsonDataOption(props, 'password')}
            />
          </Field>
        </Stack>
      </ConfigSection>

      <Divider />

      <ConfigSection title="Additional settings" isCollapsible>
        <Stack direction="column" gap={2}>
          <Field
            label="Min time interval"
            description="A lower limit for the auto group by time interval. Recommended to be set to write frequency."
          >
            <Input
              placeholder="1m"
              value={jsonData.timeInterval || ''}
              onChange={onUpdateDatasourceJsonDataOption(props, 'timeInterval')}
              width={WIDTH_LONG}
            />
          </Field>

          <Field label="TLS Skip Verify" description="Skip TLS certificate verification">
            <Switch
              value={jsonData.tlsSkipVerify || false}
              onChange={onTLSSkipVerifyChanged}
            />
          </Field>

          <MaxOpenConnectionsField
            labelWidth={WIDTH_LONG}
            jsonData={jsonData}
            onMaxConnectionsChanged={onMaxConnectionsChanged}
          />
          <MaxLifetimeField
            labelWidth={WIDTH_LONG}
            jsonData={jsonData}
            onMaxLifetimeChanged={onMaxLifetimeChanged}
          />
        </Stack>
      </ConfigSection>
    </>
  );
};
