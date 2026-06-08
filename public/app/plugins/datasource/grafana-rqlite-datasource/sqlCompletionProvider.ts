import {
  type ColumnDefinition,
  getStandardSQLCompletionProvider,
  type LanguageCompletionProvider,
  type TableDefinition,
  type TableIdentifier,
} from '@grafana/plugin-ui';
import { type DB, type SQLQuery } from '@grafana/sql';

interface CompletionProviderGetterArgs {
  getColumns: React.MutableRefObject<(t: SQLQuery) => Promise<ColumnDefinition[]>>;
  getTables: React.MutableRefObject<() => Promise<TableDefinition[]>>;
}

export const getSqlCompletionProvider: (args: CompletionProviderGetterArgs) => LanguageCompletionProvider =
  ({ getColumns, getTables }) =>
  (monaco, language) => ({
    ...(language && getStandardSQLCompletionProvider(monaco, language)),
    tables: {
      resolve: async () => {
        return await getTables.current();
      },
    },
    columns: {
      resolve: async (t?: TableIdentifier) => {
        return await getColumns.current({ table: t?.table, refId: 'A' });
      },
    },
  });

export async function fetchColumns(db: DB, q: SQLQuery): Promise<ColumnDefinition[]> {
  const cols = await db.fields(q);
  if (cols.length > 0) {
    return cols.map((c) => ({
      name: c.value,
      type: c.type ?? c.value,
      description: c.label ?? c.value,
    }));
  }
  return [];
}

export async function fetchTables(db: DB): Promise<TableDefinition[]> {
  const tables = await db.lookup?.();
  return tables || [];
}
