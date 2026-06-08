import { DataSourcePlugin } from '@grafana/data';
import { SqlQueryEditorLazy, type SQLQuery } from '@grafana/sql';

import { RqliteConfigEditor } from './configuration/ConfigurationEditor';
import { RqliteDatasource } from './datasource';
import { type RqliteOptions, type SecureJsonData } from './types';

export const plugin = new DataSourcePlugin<RqliteDatasource, SQLQuery, RqliteOptions, SecureJsonData>(
  RqliteDatasource
)
  .setQueryEditor(SqlQueryEditorLazy)
  .setConfigEditor(RqliteConfigEditor);
