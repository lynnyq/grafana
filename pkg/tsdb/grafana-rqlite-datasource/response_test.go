package rqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// ============================================================================
// fieldTypeFromRqliteType tests
// ============================================================================

func TestFieldTypeFromRqliteType(t *testing.T) {
	tests := []struct {
		name       string
		rqliteType string
		colName    string
		expected   data.FieldType
	}{
		// Integer types
		{"INTEGER", "INTEGER", "id", data.FieldTypeNullableInt64},
		{"integer lowercase", "integer", "id", data.FieldTypeNullableInt64},
		{"INT", "INT", "id", data.FieldTypeNullableInt64},
		{"BIGINT", "BIGINT", "id", data.FieldTypeNullableInt64},
		{"SMALLINT", "SMALLINT", "id", data.FieldTypeNullableInt64},
		{"TINYINT", "TINYINT", "id", data.FieldTypeNullableInt64},

		// Real/Float types
		{"REAL", "REAL", "score", data.FieldTypeNullableFloat64},
		{"real lowercase", "real", "score", data.FieldTypeNullableFloat64},
		{"FLOAT", "FLOAT", "score", data.FieldTypeNullableFloat64},
		{"DOUBLE", "DOUBLE", "score", data.FieldTypeNullableFloat64},
		{"NUMERIC", "NUMERIC", "score", data.FieldTypeNullableFloat64},
		{"DECIMAL", "DECIMAL", "score", data.FieldTypeNullableFloat64},

		// Boolean types
		{"BOOLEAN", "BOOLEAN", "active", data.FieldTypeNullableBool},
		{"BOOL", "BOOL", "active", data.FieldTypeNullableBool},

		// Time types
		{"DATETIME", "DATETIME", "some_col", data.FieldTypeNullableTime},
		{"DATE type", "DATE", "some_col", data.FieldTypeNullableTime},

		// Text types (default)
		{"TEXT", "TEXT", "name", data.FieldTypeNullableString},
		{"text lowercase", "text", "name", data.FieldTypeNullableString},
		{"VARCHAR", "VARCHAR", "name", data.FieldTypeNullableString},
		{"BLOB", "BLOB", "data", data.FieldTypeNullableString},
		{"NULL type", "NULL", "data", data.FieldTypeNullableString},
		{"empty", "", "name", data.FieldTypeNullableString},
		{"unknown", "UNKNOWN_TYPE", "name", data.FieldTypeNullableString},

		// Time column auto-detection by name (SQLite has no native time type)
		{"time column name", "INTEGER", "time", data.FieldTypeNullableTime},
		{"ts column name", "INTEGER", "ts", data.FieldTypeNullableTime},
		{"timestamp column name", "TEXT", "timestamp", data.FieldTypeNullableTime},
		{"created_at column name", "INTEGER", "created_at", data.FieldTypeNullableTime},
		{"updated_at column name", "INTEGER", "updated_at", data.FieldTypeNullableTime},
		{"deleted_at column name", "INTEGER", "deleted_at", data.FieldTypeNullableTime},
		{"date column name", "TEXT", "date", data.FieldTypeNullableTime},
		{"datetime column name", "TEXT", "datetime", data.FieldTypeNullableTime},
		{"Time uppercase", "INTEGER", "Time", data.FieldTypeNullableTime},
		{"non-time column with INTEGER", "INTEGER", "count", data.FieldTypeNullableInt64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fieldTypeFromRqliteType(tt.rqliteType, tt.colName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// toNullableInt64 tests
// ============================================================================

func TestToNullableInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *int64
	}{
		{"nil returns nil", nil, nil},
		{"float64 42", float64(42), int64Ptr(42)},
		{"float64 0", float64(0), int64Ptr(0)},
		{"float64 -1", float64(-1), int64Ptr(-1)},
		{"int64 100", int64(100), int64Ptr(100)},
		{"string '42'", "42", int64Ptr(42)},
		{"string '-100'", "-100", int64Ptr(-100)},
		{"string invalid", "not_a_number", nil},
		{"bool true", true, int64Ptr(1)},
		{"bool false", false, int64Ptr(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toNullableInt64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// toNullableFloat64 tests
// ============================================================================

func TestToNullableFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *float64
	}{
		{"nil returns nil", nil, nil},
		{"float64 3.14", float64(3.14), float64Ptr(3.14)},
		{"float64 0.0", float64(0.0), float64Ptr(0.0)},
		{"float64 -1.5", float64(-1.5), float64Ptr(-1.5)},
		{"int64 100", int64(100), float64Ptr(100.0)},
		{"string '3.14'", "3.14", float64Ptr(3.14)},
		{"string invalid", "not_a_number", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toNullableFloat64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// toNullableBool tests
// ============================================================================

func TestToNullableBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *bool
	}{
		{"nil returns nil", nil, nil},
		{"bool true", true, boolPtr(true)},
		{"bool false", false, boolPtr(false)},
		{"float64 1", float64(1), boolPtr(true)},
		{"float64 0", float64(0), boolPtr(false)},
		{"float64 0.5", float64(0.5), boolPtr(true)},
		{"int64 1", int64(1), boolPtr(true)},
		{"int64 0", int64(0), boolPtr(false)},
		{"string 'true'", "true", boolPtr(true)},
		{"string 'false'", "false", boolPtr(false)},
		{"string '1'", "1", boolPtr(true)},
		{"string '0'", "0", boolPtr(false)},
		{"string invalid", "maybe", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toNullableBool(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// toNullableTime tests - CRITICAL for frontend-backend compatibility
// ============================================================================

func TestToNullableTime(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *time.Time
	}{
		// Unix timestamp in seconds (float64)
		{
			name:     "float64 unix seconds",
			input:    float64(1718000000),
			expected: timePtr(time.Unix(1718000000, 0).UTC()),
		},
		// Unix timestamp 0
		{
			name:     "float64 unix epoch zero",
			input:    float64(0),
			expected: timePtr(time.Unix(0, 0).UTC()),
		},
		// Unix timestamp in milliseconds (float64 > 1e12)
		{
			name:     "float64 unix milliseconds",
			input:    float64(1718000000000),
			expected: timePtr(time.Unix(0, 1718000000000*int64(time.Millisecond)).UTC()),
		},
		// Unix timestamp in seconds (int64)
		{
			name:     "int64 unix seconds",
			input:    int64(1718000000),
			expected: timePtr(time.Unix(1718000000, 0).UTC()),
		},
		// Unix timestamp in milliseconds (int64 > 1e12)
		{
			name:     "int64 unix milliseconds",
			input:    int64(1718000000000),
			expected: timePtr(time.Unix(0, 1718000000000*int64(time.Millisecond)).UTC()),
		},
		// String Unix timestamp
		{
			name:     "string unix seconds",
			input:    "1718000000",
			expected: timePtr(time.Unix(1718000000, 0).UTC()),
		},
		// String RFC3339
		{
			name:     "string RFC3339",
			input:    "2024-06-10T12:00:00Z",
			expected: timePtr(time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)),
		},
		// String RFC3339 with timezone
		{
			name:     "string RFC3339 with timezone",
			input:    "2024-06-10T20:00:00+08:00",
			expected: timePtr(time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)),
		},
		// nil
		{
			name:     "nil returns nil",
			input:    nil,
			expected: nil,
		},
		// Invalid string
		{
			name:     "string invalid",
			input:    "not-a-time",
			expected: nil,
		},
		// Float64 with fractional seconds
		{
			name:     "float64 unix seconds with fraction",
			input:    float64(1718000000.5),
			expected: timePtr(time.Unix(1718000000, 500000000).UTC()),
		},
		// SQLite DATETIME format: "2024-06-10 12:00:00"
		{
			name:     "string SQLite DATETIME",
			input:    "2024-06-10 12:00:00",
			expected: timePtr(time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)),
		},
		// SQLite DATETIME with fractional seconds
		{
			name:     "string SQLite DATETIME with fractional seconds",
			input:    "2024-06-10 12:00:00.123456",
			expected: timePtr(time.Date(2024, 6, 10, 12, 0, 0, 123456000, time.UTC)),
		},
		// ISO 8601 without timezone
		{
			name:     "string ISO 8601 without timezone",
			input:    "2024-06-10T12:00:00",
			expected: timePtr(time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)),
		},
		// Date only
		{
			name:     "string date only",
			input:    "2024-06-10",
			expected: timePtr(time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)),
		},
		// SQLite DATETIME without seconds
		{
			name:     "string SQLite DATETIME without seconds",
			input:    "2024-06-10 12:00",
			expected: timePtr(time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)),
		},
		// time.Time value (gorqlite Slice() converts DATETIME columns to time.Time)
		{
			name:     "time.Time value from gorqlite",
			input:    time.Date(2024, 6, 10, 12, 30, 45, 0, time.UTC),
			expected: timePtr(time.Date(2024, 6, 10, 12, 30, 45, 0, time.UTC)),
		},
		// *time.Time value
		{
			name:     "*time.Time pointer from gorqlite",
			input:    timePtr(time.Date(2024, 6, 10, 12, 30, 45, 0, time.UTC)),
			expected: timePtr(time.Date(2024, 6, 10, 12, 30, 45, 0, time.UTC)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toNullableTime(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			// Compare with second precision (some conversions lose sub-second precision)
			assert.Equal(t, tt.expected.UTC(), result.UTC())
		})
	}
}

// ============================================================================
// toNullableString tests
// ============================================================================

func TestToNullableString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *string
	}{
		{"string", "hello", strPtr("hello")},
		{"float64", float64(42.5), strPtr("42.5")},
		{"float64 integer value", float64(42), strPtr("42")},
		{"int64", int64(100), strPtr("100")},
		{"bool true", true, strPtr("1")},
		{"bool false", false, strPtr("0")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toNullableString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// jsonResultToFrame tests - end-to-end DataFrame conversion
// ============================================================================

func TestJsonResultToFrame_AllSQLiteDataTypes(t *testing.T) {
	// Simulates a rqlite query result with all SQLite data types
	// This is the exact JSON format rqlite returns from /db/query
	result := &rqliteJSONResult{
		Columns: []string{"id", "name", "score", "active", "created_at", "data", "null_col"},
		Types:   []string{"INTEGER", "TEXT", "REAL", "BOOLEAN", "INTEGER", "BLOB", "TEXT"},
		Values: [][]interface{}{
			{float64(1), "Alice", float64(95.5), true, float64(1718000000), "blob_data", nil},
			{float64(2), "Bob", float64(87.3), false, float64(1718001000), nil, nil},
			{nil, nil, nil, nil, nil, nil, nil},
		},
	}

	frame, err := jsonResultToFrame(result, "test", 1000000)
	require.NoError(t, err)
	require.NotNil(t, frame)

	// Verify field count
	assert.Len(t, frame.Fields, 7)

	// Field 0: id (INTEGER → NullableInt64)
	assert.Equal(t, "id", frame.Fields[0].Name)
	assert.Equal(t, data.FieldTypeNullableInt64, frame.Fields[0].Type())
	assert.Equal(t, int64Ptr(1), frame.Fields[0].At(0))
	assert.Equal(t, int64Ptr(2), frame.Fields[0].At(1))
	assert.Nil(t, frame.Fields[0].At(2))

	// Field 1: name (TEXT → NullableString)
	assert.Equal(t, "name", frame.Fields[1].Name)
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[1].Type())
	assert.Equal(t, strPtr("Alice"), frame.Fields[1].At(0))
	assert.Equal(t, strPtr("Bob"), frame.Fields[1].At(1))
	assert.Nil(t, frame.Fields[1].At(2))

	// Field 2: score (REAL → NullableFloat64)
	assert.Equal(t, "score", frame.Fields[2].Name)
	assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[2].Type())
	assert.Equal(t, float64Ptr(95.5), frame.Fields[2].At(0))
	assert.Equal(t, float64Ptr(87.3), frame.Fields[2].At(1))
	assert.Nil(t, frame.Fields[2].At(2))

	// Field 3: active (BOOLEAN → NullableBool)
	assert.Equal(t, "active", frame.Fields[3].Name)
	assert.Equal(t, data.FieldTypeNullableBool, frame.Fields[3].Type())
	assert.Equal(t, boolPtr(true), frame.Fields[3].At(0))
	assert.Equal(t, boolPtr(false), frame.Fields[3].At(1))
	assert.Nil(t, frame.Fields[3].At(2))

	// Field 4: created_at (INTEGER → auto-detected as NullableTime by column name)
	// SQLite has no native time type, so "created_at" is auto-detected as time
	assert.Equal(t, "created_at", frame.Fields[4].Name)
	assert.Equal(t, data.FieldTypeNullableTime, frame.Fields[4].Type())
	// Verify the time value is correctly converted from Unix timestamp
	createdAt0 := frame.Fields[4].At(0)
	require.NotNil(t, createdAt0)
	t0, ok := createdAt0.(*time.Time)
	require.True(t, ok)
	assert.Equal(t, time.Unix(1718000000, 0).UTC(), t0.UTC())
	assert.Nil(t, frame.Fields[4].At(2))

	// Field 5: data (BLOB → NullableString)
	assert.Equal(t, "data", frame.Fields[5].Name)
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[5].Type())
	assert.Equal(t, strPtr("blob_data"), frame.Fields[5].At(0))
	assert.Nil(t, frame.Fields[5].At(1))
	assert.Nil(t, frame.Fields[5].At(2))

	// Field 6: null_col (TEXT → NullableString, all nil)
	assert.Equal(t, "null_col", frame.Fields[6].Name)
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[6].Type())
	assert.Nil(t, frame.Fields[6].At(0))
	assert.Nil(t, frame.Fields[6].At(1))
	assert.Nil(t, frame.Fields[6].At(2))
}

func TestJsonResultToFrame_TimeColumnAsDatetime(t *testing.T) {
	// When rqlite returns a column typed as DATETIME, it should map to NullableTime
	result := &rqliteJSONResult{
		Columns: []string{"time", "value"},
		Types:   []string{"DATETIME", "REAL"},
		Values: [][]interface{}{
			{float64(1718000000), float64(42.0)},
			{float64(1718001000), float64(43.5)},
		},
	}

	frame, err := jsonResultToFrame(result, "time_test", 1000000)
	require.NoError(t, err)
	require.NotNil(t, frame)

	// Field 0: time (DATETIME → NullableTime)
	assert.Equal(t, "time", frame.Fields[0].Name)
	assert.Equal(t, data.FieldTypeNullableTime, frame.Fields[0].Type())

	time0 := frame.Fields[0].At(0)
	require.NotNil(t, time0)
	timeVal0, ok := time0.(*time.Time)
	require.True(t, ok, "expected *time.Time, got %T", time0)
	assert.Equal(t, time.Unix(1718000000, 0).UTC(), timeVal0.UTC())

	time1 := frame.Fields[0].At(1)
	require.NotNil(t, time1)
	timeVal1, ok := time1.(*time.Time)
	require.True(t, ok, "expected *time.Time, got %T", time1)
	assert.Equal(t, time.Unix(1718001000, 0).UTC(), timeVal1.UTC())

	// Field 1: value (REAL → NullableFloat64)
	assert.Equal(t, "value", frame.Fields[1].Name)
	assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[1].Type())
}

func TestJsonResultToFrame_TimeColumnUnixMs(t *testing.T) {
	// Unix timestamp in milliseconds (> 1e12)
	result := &rqliteJSONResult{
		Columns: []string{"time", "value"},
		Types:   []string{"DATETIME", "REAL"},
		Values: [][]interface{}{
			{float64(1718000000000), float64(42.0)},
		},
	}

	frame, err := jsonResultToFrame(result, "ms_test", 1000000)
	require.NoError(t, err)

	time0 := frame.Fields[0].At(0)
	require.NotNil(t, time0)
	timeVal, ok := time0.(*time.Time)
	require.True(t, ok)
	assert.Equal(t, time.Unix(1718000000, 0).UTC(), timeVal.UTC())
}

func TestJsonResultToFrame_TimeColumnRFC3339String(t *testing.T) {
	// RFC3339 string in a DATETIME column
	result := &rqliteJSONResult{
		Columns: []string{"time", "value"},
		Types:   []string{"DATETIME", "REAL"},
		Values: [][]interface{}{
			{"2024-06-10T12:00:00Z", float64(42.0)},
		},
	}

	frame, err := jsonResultToFrame(result, "rfc_test", 1000000)
	require.NoError(t, err)

	time0 := frame.Fields[0].At(0)
	require.NotNil(t, time0)
	timeVal, ok := time0.(*time.Time)
	require.True(t, ok)
	assert.Equal(t, time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC), timeVal.UTC())
}

func TestJsonResultToFrame_EmptyResult(t *testing.T) {
	tests := []struct {
		name   string
		result *rqliteJSONResult
	}{
		{"nil result", nil},
		{"empty columns", &rqliteJSONResult{Columns: []string{}}},
		{"no values", &rqliteJSONResult{Columns: []string{"id"}, Types: []string{"INTEGER"}, Values: nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := jsonResultToFrame(tt.result, "empty", 1000000)
			require.NoError(t, err)
			require.NotNil(t, frame)
			assert.Len(t, frame.Fields, 0)
		})
	}
}

func TestJsonResultToFrame_RowLimit(t *testing.T) {
	result := &rqliteJSONResult{
		Columns: []string{"id", "name"},
		Types:   []string{"INTEGER", "TEXT"},
		Values: [][]interface{}{
			{float64(1), "Alice"},
			{float64(2), "Bob"},
			{float64(3), "Charlie"},
			{float64(4), "Dave"},
			{float64(5), "Eve"},
		},
	}

	frame, err := jsonResultToFrame(result, "limited", 3)
	require.NoError(t, err)
	require.NotNil(t, frame)

	// Should only have 3 rows
	assert.Equal(t, 3, frame.Fields[0].Len())

	// Should have a warning notice
	require.NotNil(t, frame.Meta)
	require.Len(t, frame.Meta.Notices, 1)
	assert.Equal(t, data.NoticeSeverityWarning, frame.Meta.Notices[0].Severity)
	assert.Contains(t, frame.Meta.Notices[0].Text, "limited")
}

func TestJsonResultToFrame_NullValues(t *testing.T) {
	result := &rqliteJSONResult{
		Columns: []string{"int_col", "float_col", "str_col", "bool_col"},
		Types:   []string{"INTEGER", "REAL", "TEXT", "BOOLEAN"},
		Values: [][]interface{}{
			{nil, nil, nil, nil},
		},
	}

	frame, err := jsonResultToFrame(result, "nulls", 1000000)
	require.NoError(t, err)
	require.NotNil(t, frame)

	assert.Nil(t, frame.Fields[0].At(0)) // int null
	assert.Nil(t, frame.Fields[1].At(0)) // float null
	assert.Nil(t, frame.Fields[2].At(0)) // string null
	assert.Nil(t, frame.Fields[3].At(0)) // bool null
}

func TestJsonResultToFrame_NoTypes(t *testing.T) {
	// When rqlite doesn't provide types, they should be inferred from actual data values
	result := &rqliteJSONResult{
		Columns: []string{"id", "name"},
		Values: [][]interface{}{
			{float64(1), "Alice"},
		},
	}

	frame, err := jsonResultToFrame(result, "no_types", 1000000)
	require.NoError(t, err)
	require.NotNil(t, frame)

	// Types should be inferred from values: float64 → REAL → NullableFloat64, string → TEXT → NullableString
	assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[0].Type())
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[1].Type())
}

func TestJsonResultToFrame_MissingColumnValues(t *testing.T) {
	// Row with fewer values than columns
	result := &rqliteJSONResult{
		Columns: []string{"id", "name", "score"},
		Types:   []string{"INTEGER", "TEXT", "REAL"},
		Values: [][]interface{}{
			{float64(1)}, // missing name and score
		},
	}

	frame, err := jsonResultToFrame(result, "missing", 1000000)
	require.NoError(t, err)
	require.NotNil(t, frame)

	assert.Equal(t, int64Ptr(1), frame.Fields[0].At(0))
	assert.Nil(t, frame.Fields[1].At(0)) // missing → nil
	assert.Nil(t, frame.Fields[2].At(0)) // missing → nil
}

// ============================================================================
// Frontend-Backend compatibility: Time type round-trip test
// ============================================================================

func TestTimeTypeFrontendBackendCompatibility(t *testing.T) {
	// This test verifies that time values can round-trip correctly between
	// the backend (Go) and frontend (TypeScript/JavaScript).
	//
	// Frontend expectations:
	//   - Grafana frontend expects time.Time fields to be serialized as
	//     RFC3339Nano strings in the JSON response
	//   - The SDK handles this automatically via data.Frame JSON marshaling
	//   - Frontend JavaScript Date objects work with millisecond precision
	//
	// Backend expectations:
	//   - rqlite stores timestamps as Unix epoch integers (seconds)
	//   - The backend converts these to time.Time for NullableTime fields
	//   - The SDK serializes time.Time → RFC3339 → frontend

	// Simulate a typical time series query result from rqlite
	// where time is stored as Unix epoch seconds in an INTEGER column
	// Note: The "time" column is auto-detected as NullableTime by column name
	// (SQLite has no native time type, so we detect by name like frser-sqlite)
	result := &rqliteJSONResult{
		Columns: []string{"time", "metric", "value"},
		Types:   []string{"INTEGER", "TEXT", "REAL"},
		Values: [][]interface{}{
			{float64(1718000000), "cpu", float64(85.5)},
			{float64(1718000060), "cpu", float64(90.2)},
			{float64(1718000120), "cpu", float64(78.1)},
		},
	}

	frame, err := jsonResultToFrame(result, "time_series", 1000000)
	require.NoError(t, err)

	// Verify the time column is auto-detected as NullableTime (by column name "time")
	timeField := frame.Fields[0]
	assert.Equal(t, "time", timeField.Name)
	assert.Equal(t, data.FieldTypeNullableTime, timeField.Type())

	// Verify values are correctly converted to time.Time from Unix timestamps
	time0Val := timeField.At(0)
	require.NotNil(t, time0Val)
	time0, ok := time0Val.(*time.Time)
	require.True(t, ok)
	assert.Equal(t, time.Unix(1718000000, 0).UTC(), time0.UTC())
}

func TestTimeTypeExplicitDatetimeColumn(t *testing.T) {
	// When a column is explicitly typed as DATETIME, it maps to NullableTime
	// This is the case when users store timestamps in a DATETIME column
	result := &rqliteJSONResult{
		Columns: []string{"time", "value"},
		Types:   []string{"DATETIME", "REAL"},
		Values: [][]interface{}{
			{float64(1718000000), float64(85.5)},
		},
	}

	frame, err := jsonResultToFrame(result, "datetime_test", 1000000)
	require.NoError(t, err)

	timeField := frame.Fields[0]
	assert.Equal(t, data.FieldTypeNullableTime, timeField.Type())

	// Verify the time value is correctly converted
	timeVal, ok := timeField.At(0).(*time.Time)
	require.True(t, ok)
	expectedTime := time.Unix(1718000000, 0).UTC()
	assert.Equal(t, expectedTime, timeVal.UTC())
}

func TestJsonResultToFrame_GorqliteDatetimeAsTimeTime(t *testing.T) {
	// gorqlite's Slice() method automatically converts DATETIME columns to time.Time.
	// This test simulates that behavior: when gorqlite returns time.Time values
	// for a DATETIME column, the conversion must handle them correctly.
	result := &rqliteJSONResult{
		Columns: []string{"datetime", "value"},
		Types:   []string{"DATETIME", "REAL"},
		Values: [][]interface{}{
			{time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC), float64(42.0)},
			{time.Date(2024, 6, 10, 13, 30, 0, 0, time.UTC), float64(55.5)},
			{nil, nil},
		},
	}

	frame, err := jsonResultToFrame(result, "gorqlite_datetime", 1000000)
	require.NoError(t, err)
	require.NotNil(t, frame)

	// Field 0: datetime (DATETIME → NullableTime)
	assert.Equal(t, "datetime", frame.Fields[0].Name)
	assert.Equal(t, data.FieldTypeNullableTime, frame.Fields[0].Type())

	// Verify time.Time values are preserved correctly
	time0 := frame.Fields[0].At(0)
	require.NotNil(t, time0)
	timeVal0, ok := time0.(*time.Time)
	require.True(t, ok, "expected *time.Time, got %T", time0)
	assert.Equal(t, time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC), timeVal0.UTC())

	time1 := frame.Fields[0].At(1)
	require.NotNil(t, time1)
	timeVal1, ok := time1.(*time.Time)
	require.True(t, ok, "expected *time.Time, got %T", time1)
	assert.Equal(t, time.Date(2024, 6, 10, 13, 30, 0, 0, time.UTC), timeVal1.UTC())

	// nil value stays nil
	assert.Nil(t, frame.Fields[0].At(2))
}

// ============================================================================
// Helper functions
// ============================================================================

func int64Ptr(v int64) *int64    { return &v }
func float64Ptr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool       { return &v }
func strPtr(v string) *string    { return &v }
func timePtr(v time.Time) *time.Time { return &v }
