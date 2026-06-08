import { type ScopedVars, type VariableWithMultiSupport } from '@grafana/data';
import { type TemplateSrv } from '@grafana/runtime';
import { type SqlQueryModel, type SQLQuery } from '@grafana/sql';

export class RqliteQueryModel implements SqlQueryModel {
  constructor(
    private target?: SQLQuery,
    private templateSrv?: TemplateSrv,
    private scopedVars?: ScopedVars
  ) {}

  quoteLiteral(value: string): string {
    return "'" + String(value).replace(/'/g, "''") + "'";
  }
}
