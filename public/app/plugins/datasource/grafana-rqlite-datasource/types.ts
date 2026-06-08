import { type SQLOptions } from '@grafana/sql';

export enum ConnectionMode {
  AutoDiscovery = 'auto-discovery',
  StaticList = 'static-list',
}

export enum ReadConsistency {
  None = 'none',
  Weak = 'weak',
  Strong = 'strong',
  Linearizable = 'linearizable',
}

export interface RqliteOptions extends SQLOptions {
  clusterUrls?: string;
  connectionMode?: ConnectionMode;
  readConsistency?: ReadConsistency;
  tlsSkipVerify?: boolean;
}

export interface SecureJsonData {
  password?: string;
}
