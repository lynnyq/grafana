import {
  type DataSourceInstanceSettings,
  type ScopedVars,
  type VariableWithMultiSupport,
  generateUUID,
} from '@grafana/data';
import { type LanguageDefinition } from '@grafana/plugin-ui';
import { type TemplateSrv } from '@grafana/runtime';
import {
  COMMON_FNS,
  type DB,
  type FuncParameter,
  MACRO_FUNCTIONS,
  type SQLQuery,
  type SQLSelectableValue,
  SqlDatasource,
  SQLVariableSupport,
  formatSQL,
} from '@grafana/sql';

import { RqliteQueryModel } from './RqliteQueryModel';
import { getSchema, getVersion, showTables } from './rqliteMetaQuery';
import { getSqlCompletionProvider, fetchColumns, fetchTables } from './sqlCompletionProvider';
import { getFieldConfig, toRawSql } from './sqlUtil';
import { type RqliteOptions } from './types';

export class RqliteDatasource extends SqlDatasource {
  sqlLanguageDefinition: LanguageDefinition | undefined = undefined;

  constructor(instanceSettings: DataSourceInstanceSettings<RqliteOptions>) {
    super(instanceSettings);
    this.dialect = 'other';
    this.variables = new SQLVariableSupport(this);
    // rqlite is SQLite-based with a single default database named 'main'.
    // Setting preconfiguredDatabase ensures the DatasetSelector auto-selects it,
    // which in turn allows the TableSelector to fetch tables.
    this.preconfiguredDatabase = 'main';
  }

  getQueryModel(target?: SQLQuery, templateSrv?: TemplateSrv, scopedVars?: ScopedVars): RqliteQueryModel {
    return new RqliteQueryModel(target, templateSrv, scopedVars);
  }

  interpolateVariable = (value: string | string[] | number, variable: VariableWithMultiSupport) => {
    if (typeof value === 'string') {
      return String(value).replace(/'/g, "''");
    }

    if (typeof value === 'number') {
      return value;
    }

    if (Array.isArray(value)) {
      const quotedValues = value.map((v) => this.getQueryModel().quoteLiteral(v));
      return '(' + quotedValues.join(',') + ')';
    }

    return value;
  };

  async getVersion(): Promise<string> {
    const value = await this.runSql<{ version: string }>(getVersion());
    const results = value.fields.version?.values;

    if (!results) {
      return '';
    }

    return results[0].toString();
  }

  async fetchTables(): Promise<string[]> {
    const tables = await this.runSql<{ name: string }>(showTables(), { refId: 'tables' });
    return tables.fields.name?.values.flat() ?? [];
  }

  getSqlLanguageDefinition(db: DB): LanguageDefinition {
    if (this.sqlLanguageDefinition !== undefined) {
      return this.sqlLanguageDefinition;
    }

    const args = {
      getColumns: { current: (query: SQLQuery) => fetchColumns(db, query) },
      getTables: { current: () => fetchTables(db) },
    };
    this.sqlLanguageDefinition = {
      id: 'sql',
      completionProvider: getSqlCompletionProvider(args),
      formatter: formatSQL,
    };
    return this.sqlLanguageDefinition;
  }

  async fetchFields(query: SQLQuery): Promise<SQLSelectableValue[]> {
    const { table } = query;
    if (table === undefined) {
      return [];
    }
    const schema = await this.runSql<{ name: string; type: string }>(getSchema(table), {
      refId: `columns-${generateUUID()}`,
    });
    const result: SQLSelectableValue[] = [];
    for (let i = 0; i < schema.length; i++) {
      const column = schema.fields.name.values[i];
      const type = schema.fields.type.values[i];
      result.push({ label: column, value: column, type, ...getFieldConfig(type) });
    }
    return result;
  }

  getFunctions = (): ReturnType<DB['functions']> => {
    const columnParam: FuncParameter = {
      name: 'Column',
      required: true,
      options: (query) => this.fetchFields(query),
    };

    return [...MACRO_FUNCTIONS(columnParam), ...COMMON_FNS.map((fn) => ({ ...fn, parameters: [columnParam] }))];
  };

  getDB(): DB {
    if (this.db !== undefined) {
      return this.db;
    }

    return {
      init: () => Promise.resolve(true),
      datasets: () => Promise.resolve(['main']),
      tables: (_dataset?: string) => this.fetchTables(),
      getEditorLanguageDefinition: () => this.getSqlLanguageDefinition(this.db),
      fields: async (query: SQLQuery) => {
        if (!query?.table) {
          return [];
        }
        return this.fetchFields(query);
      },
      validateQuery: (query) =>
        Promise.resolve({ isError: false, isValid: true, query, error: '', rawSql: query.rawSql }),
      toRawSql,
      functions: () => this.getFunctions(),
      lookup: async () => {
        const tables = await this.fetchTables();
        return tables.map((t) => ({ name: t, completion: t }));
      },
    };
  }
}
