import { isEmpty } from 'lodash';

import { createSelectClause, haveColumns, type RAQBFieldTypes, type SQLQuery } from '@grafana/sql';

export function getFieldConfig(type: string): { raqbFieldType: RAQBFieldTypes } {
  switch (type.toUpperCase()) {
    case 'INTEGER':
    case 'INT':
    case 'BIGINT':
    case 'SMALLINT':
    case 'TINYINT':
      return { raqbFieldType: 'number' };
    case 'REAL':
    case 'FLOAT':
    case 'DOUBLE':
    case 'NUMERIC':
    case 'DECIMAL':
      return { raqbFieldType: 'number' };
    case 'TEXT':
    case 'VARCHAR':
    case 'CHAR':
    case 'NULL':
    case 'BLOB':
      return { raqbFieldType: 'text' };
    case 'BOOLEAN':
    case 'BOOL':
      return { raqbFieldType: 'boolean' };
    case 'DATETIME':
    case 'DATE':
      return { raqbFieldType: 'datetime' };
    default:
      return { raqbFieldType: 'text' };
  }
}

export function toRawSql({ sql, table }: SQLQuery): string {
  let rawQuery = '';

  // Return early with empty string if there is no sql column
  if (!sql || !haveColumns(sql.columns)) {
    return rawQuery;
  }

  rawQuery += createSelectClause(sql.columns);

  if (table) {
    rawQuery += `FROM ${table} `;
  }

  if (sql.whereString) {
    rawQuery += `WHERE ${sql.whereString} `;
  }

  if (sql.groupBy?.[0]?.property.name) {
    const groupBy = sql.groupBy.map((g) => g.property.name).filter((g) => !isEmpty(g));
    rawQuery += `GROUP BY ${groupBy.join(', ')} `;
  }

  if (sql.orderBy?.property.name) {
    rawQuery += `ORDER BY ${sql.orderBy.property.name} `;
  }

  if (sql.orderBy?.property.name && sql.orderByDirection) {
    rawQuery += `${sql.orderByDirection} `;
  }

  if (sql.limit !== undefined && sql.limit >= 0) {
    rawQuery += `LIMIT ${sql.limit} `;
  }

  return rawQuery;
}
