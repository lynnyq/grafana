import { of } from 'rxjs';

import {
  type DataSourceInstanceSettings,
  FieldType,
  dataFrameToJSON,
  createDataFrame,
} from '@grafana/data';
import { getBackendSrv, setBackendSrv, type BackendSrv, type FetchResponse } from '@grafana/runtime';

import { RqliteDatasource } from './datasource';
import { ConnectionMode, type RqliteOptions, ReadConsistency } from './types';

const backendSrv: BackendSrv = {
  fetch: () => undefined,
} as unknown as BackendSrv;

const uid = '0000';
jest.mock('@grafana/data', () => ({
  ...jest.requireActual('@grafana/data'),
  generateUUID: () => uid,
}));

let origBackendSrv: BackendSrv;
beforeAll(() => {
  origBackendSrv = getBackendSrv();
  setBackendSrv(backendSrv);
});

afterAll(() => {
  setBackendSrv(origBackendSrv);
});

function createFetchResponse<T>(data: T): FetchResponse<T> {
  return {
    data,
    status: 200,
    statusText: 'OK',
    ok: true,
    headers: new Headers(),
    url: '',
    type: 'basic',
    redirected: false,
  };
}

// Creates a mock backend response in the format /api/ds/query returns
function createQueryResponse(refId: string, frames: Array<ReturnType<typeof dataFrameToJSON>>) {
  return {
    results: {
      [refId]: {
        refId,
        frames,
      },
    },
  };
}

describe('RqliteDatasource', () => {
  const fetchMock = jest.spyOn(backendSrv, 'fetch');

  const createDatasource = () => {
    const instanceSettings = {
      jsonData: {
        connectionMode: ConnectionMode.AutoDiscovery,
        readConsistency: ReadConsistency.Weak,
      },
      url: 'http://localhost:4001',
    } as unknown as DataSourceInstanceSettings<RqliteOptions>;
    return new RqliteDatasource(instanceSettings);
  };

  describe('constructor', () => {
    it('should set preconfiguredDatabase to main', () => {
      const ds = createDatasource();
      expect(ds.preconfiguredDatabase).toBe('main');
    });
  });

  describe('interpolateVariable', () => {
    const ds = createDatasource();

    it('should escape single quotes in strings', () => {
      const result = ds.interpolateVariable("O'Brien", { multi: false, includeAll: false, current: {} } as any);
      expect(result).toBe("O''Brien");
    });

    it('should return numbers as-is', () => {
      const result = ds.interpolateVariable(42, { multi: false, includeAll: false, current: {} } as any);
      expect(result).toBe(42);
    });

    it('should join arrays with commas and quote each value', () => {
      const result = ds.interpolateVariable(['a', 'b', 'c'], {
        multi: true,
        includeAll: false,
        current: {},
      } as any);
      expect(result).toBe("('a','b','c')");
    });

    it('should escape quotes in array values', () => {
      const result = ds.interpolateVariable(["it's", "they're"], {
        multi: true,
        includeAll: false,
        current: {},
      } as any);
      expect(result).toBe("('it''s','they''re')");
    });
  });

  describe('getDB', () => {
    it('should return main as the only dataset', async () => {
      const ds = createDatasource();
      const db = ds.getDB();
      const datasets = await db.datasets();
      expect(datasets).toEqual(['main']);
    });
  });

  describe('fetchTables', () => {
    it('should return list of table names', async () => {
      const frame = createDataFrame({
        refId: 'meta',
        fields: [{ name: 'name', values: ['users', 'orders', 'metrics'] }],
      });
      const response = createQueryResponse('tables', [dataFrameToJSON(frame)]);
      jest.clearAllMocks();
      fetchMock.mockImplementation(() => of(createFetchResponse(response)) as any);

      const ds = createDatasource();
      const tables = await ds.fetchTables();
      expect(tables).toEqual(['users', 'orders', 'metrics']);
    });
  });

  describe('fetchFields', () => {
    it('should return field info with types', async () => {
      const frame = createDataFrame({
        refId: 'meta',
        fields: [
          { name: 'name', values: ['id', 'name', 'score', 'created_at'] },
          { name: 'type', values: ['INTEGER', 'TEXT', 'REAL', 'INTEGER'] },
        ],
      });
      const response = createQueryResponse('columns-0000', [dataFrameToJSON(frame)]);
      jest.clearAllMocks();
      fetchMock.mockImplementation(() => of(createFetchResponse(response)) as any);

      const ds = createDatasource();
      const fields = await ds.fetchFields({ table: 'users' } as any);
      expect(fields).toHaveLength(4);
      expect(fields[0]).toEqual(expect.objectContaining({ label: 'id', value: 'id', type: 'INTEGER' }));
      expect(fields[1]).toEqual(expect.objectContaining({ label: 'name', value: 'name', type: 'TEXT' }));
      expect(fields[2]).toEqual(expect.objectContaining({ label: 'score', value: 'score', type: 'REAL' }));
      expect(fields[3]).toEqual(expect.objectContaining({ label: 'created_at', value: 'created_at', type: 'INTEGER' }));
    });

    it('should return empty array when no table specified', async () => {
      const ds = createDatasource();
      const fields = await ds.fetchFields({} as any);
      expect(fields).toEqual([]);
    });
  });

  describe('getVersion', () => {
    it('should return SQLite version string', async () => {
      const frame = createDataFrame({
        refId: 'meta',
        fields: [{ name: 'version', values: ['3.42.0'] }],
      });
      const response = createQueryResponse('meta', [dataFrameToJSON(frame)]);
      jest.clearAllMocks();
      fetchMock.mockImplementation(() => of(createFetchResponse(response)) as any);

      const ds = createDatasource();
      const version = await ds.getVersion();
      expect(version).toBe('3.42.0');
    });
  });
});

// ============================================================================
// Frontend-Backend type compatibility tests
// ============================================================================

describe('Frontend-Backend Type Compatibility', () => {
  describe('DataFrame field types from backend', () => {
    it('should correctly handle INTEGER columns as NullableInt64', () => {
      const frame = createDataFrame({
        fields: [
          { name: 'id', type: FieldType.number, values: [1, 2, 3] },
          { name: 'value', type: FieldType.number, values: [10.5, 20.3, 30.1] },
        ],
      });
      expect(frame.fields[0].type).toBe(FieldType.number);
      expect(frame.fields[1].type).toBe(FieldType.number);
    });

    it('should correctly handle TEXT columns as NullableString', () => {
      const frame = createDataFrame({
        fields: [{ name: 'name', type: FieldType.string, values: ['Alice', 'Bob', null] }],
      });
      expect(frame.fields[0].type).toBe(FieldType.string);
      expect(frame.fields[0].values[0]).toBe('Alice');
      expect(frame.fields[0].values[2]).toBeNull();
    });

    it('should correctly handle time columns from Unix timestamps', () => {
      const frame = createDataFrame({
        fields: [
          { name: 'time', type: FieldType.time, values: [new Date(1718000000000), new Date(1718001000000)] },
          { name: 'value', type: FieldType.number, values: [85.5, 90.2] },
        ],
      });
      expect(frame.fields[0].type).toBe(FieldType.time);
      expect(frame.fields[0].values[0].getTime()).toBe(1718000000000);
    });

    it('should correctly handle boolean columns', () => {
      const frame = createDataFrame({
        fields: [{ name: 'active', type: FieldType.boolean, values: [true, false, null] }],
      });
      expect(frame.fields[0].type).toBe(FieldType.boolean);
      expect(frame.fields[0].values[0]).toBe(true);
      expect(frame.fields[0].values[2]).toBeNull();
    });

    it('should handle null values in all field types', () => {
      const frame = createDataFrame({
        fields: [
          { name: 'int_col', type: FieldType.number, values: [1, null, 3] },
          { name: 'float_col', type: FieldType.number, values: [1.5, null, 3.5] },
          { name: 'str_col', type: FieldType.string, values: ['a', null, 'c'] },
          { name: 'bool_col', type: FieldType.boolean, values: [true, null, false] },
          { name: 'time_col', type: FieldType.time, values: [new Date(1718000000000), null, new Date(1718001000000)] },
        ],
      });

      expect(frame.fields[0].values[1]).toBeNull();
      expect(frame.fields[1].values[1]).toBeNull();
      expect(frame.fields[2].values[1]).toBeNull();
      expect(frame.fields[3].values[1]).toBeNull();
      expect(frame.fields[4].values[1]).toBeNull();
    });

    it('should handle time series format with integer time column', () => {
      // Backend returns time as NullableInt64 (Unix seconds)
      // Frontend expects time as FieldType.time for time series
      // The conversion happens in applyTimeSeriesFormat on the backend
      const frame = createDataFrame({
        fields: [
          { name: 'time', type: FieldType.time, values: [new Date(1718000000000), new Date(1718000060000)] },
          { name: 'cpu', type: FieldType.number, values: [85.5, 90.2] },
        ],
      });

      expect(frame.fields[0].type).toBe(FieldType.time);
      // Verify the timestamps are correct in milliseconds (JavaScript Date precision)
      expect(frame.fields[0].values[0].getTime()).toBe(1718000000000);
      expect(frame.fields[0].values[1].getTime()).toBe(1718000060000);
    });
  });
});
