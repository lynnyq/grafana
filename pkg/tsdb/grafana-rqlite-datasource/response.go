package rqlite

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rqlite/gorqlite"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// rqliteJSONResult represents a single result from the rqlite /db/query API.
// This mirrors the JSON structure returned by rqlite, enabling direct testing
// without requiring a live gorqlite connection.
type rqliteJSONResult struct {
	Columns []string        `json:"columns,omitempty"`
	Types   []string        `json:"types,omitempty"`
	Values  [][]interface{} `json:"values,omitempty"`
	Time    float64         `json:"time,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// queryResultToFrame converts a gorqlite QueryResult into a Grafana DataFrame.
func queryResultToFrame(rows *gorqlite.QueryResult, frameName string, rowLimit int64) (*data.Frame, error) {
	if rows == nil || rows.NumRows() == 0 {
		return data.NewFrame(frameName), nil
	}

	colNames := rows.Columns()
	if len(colNames) == 0 {
		return data.NewFrame(frameName), nil
	}

	// Build a rqliteJSONResult from gorqlite QueryResult
	result := &rqliteJSONResult{
		Columns: colNames,
		Types:   rows.Types(),
	}

	for rows.Next() {
		slice, err := rows.Slice()
		if err != nil {
			return nil, fmt.Errorf("error slicing row: %w", err)
		}
		row := make([]interface{}, len(slice))
		copy(row, slice)
		result.Values = append(result.Values, row)
	}

	return jsonResultToFrame(result, frameName, rowLimit)
}

// jsonResultToFrame converts a rqlite JSON result into a Grafana DataFrame.
// This is the core conversion logic, separated from gorqlite for testability.
func jsonResultToFrame(result *rqliteJSONResult, frameName string, rowLimit int64) (*data.Frame, error) {
	if result == nil || len(result.Columns) == 0 || len(result.Values) == 0 {
		return data.NewFrame(frameName), nil
	}

	colNames := result.Columns
	colTypes := result.Types

	// If rqlite didn't return column types, infer them from the actual data values.
	// rqlite's HTTP API may omit the "types" field in some cases (e.g., PRAGMA queries,
	// expressions without explicit column types).
	if len(colTypes) < len(colNames) {
		colTypes = inferColumnTypes(colNames, result.Values)
	}

	// Create typed fields based on column types from rqlite
	fields := make(data.Fields, len(colNames))
	for i, colName := range colNames {
		fieldType := fieldTypeFromRqliteType(safeGetType(colTypes, i), colName)
		fields[i] = data.NewFieldFromFieldType(fieldType, 0)
		fields[i].Name = colName
	}

	frame := data.NewFrame(frameName, fields...)

	// Iterate through rows
	for rowIdx, row := range result.Values {
		if int64(rowIdx) >= rowLimit {
			frame.AppendNotices(data.Notice{
				Severity: data.NoticeSeverityWarning,
				Text:     fmt.Sprintf("Results have been limited to %d because the SQL row limit was reached", rowLimit),
			})
			break
		}

		rowValues := make([]any, len(colNames))
		for colIdx := range colNames {
			if colIdx >= len(row) {
				rowValues[colIdx] = nil
				continue
			}
			rawVal := row[colIdx]
			colType := safeGetType(colTypes, colIdx)
			rowValues[colIdx] = convertValueForField(rawVal, colType, fields[colIdx].Type())
		}
		frame.AppendRow(rowValues...)
	}

	return frame, nil
}

// safeGetType returns the type at index i, or empty string if out of range.
func safeGetType(types []string, i int) string {
	if i < len(types) {
		return types[i]
	}
	return ""
}

// inferColumnTypes infers column types from actual data values when rqlite
// doesn't provide them. rqlite's JSON API returns all numbers as float64
// and strings as string. We scan the first non-nil value of each column
// to determine the type.
func inferColumnTypes(colNames []string, values [][]interface{}) []string {
	types := make([]string, len(colNames))
	for colIdx := range colNames {
		types[colIdx] = inferColumnType(colIdx, values)
	}
	return types
}

// inferColumnType determines the type of a column by examining its actual values.
// JSON decoding produces: float64 for numbers, string for text, bool for booleans, nil for NULL.
func inferColumnType(colIdx int, values [][]interface{}) string {
	for _, row := range values {
		if colIdx >= len(row) {
			continue
		}
		val := row[colIdx]
		if val == nil {
			continue
		}
		switch val.(type) {
		case float64:
			return "REAL"
		case string:
			return "TEXT"
		case bool:
			return "BOOLEAN"
		}
	}
	// All values are nil, default to TEXT
	return "TEXT"
}

// Common time column names in SQLite/rqlite databases.
// SQLite has no native time type, so time values are stored as INTEGER (Unix epoch)
// or TEXT (ISO 8601 strings). We auto-detect these columns by name.
var timeColumnNames = map[string]bool{
	"time": true, "ts": true, "timestamp": true, "created_at": true,
	"updated_at": true, "deleted_at": true, "date": true, "datetime": true,
}

// fieldTypeFromRqliteType maps an rqlite/SQLite column type to a Grafana field type.
// SQLite uses dynamic typing, so the declared type may not match the actual data.
func fieldTypeFromRqliteType(colType string, colName string) data.FieldType {
	// Auto-detect time columns by name (like frser-sqlite-datasource)
	// SQLite stores timestamps as INTEGER or TEXT, not as a native time type
	if timeColumnNames[strings.ToLower(colName)] {
		return data.FieldTypeNullableTime
	}

	switch strings.ToUpper(colType) {
	case "INTEGER", "INT", "BIGINT", "SMALLINT", "TINYINT":
		return data.FieldTypeNullableInt64
	case "REAL", "FLOAT", "DOUBLE", "NUMERIC", "DECIMAL":
		return data.FieldTypeNullableFloat64
	case "BOOLEAN", "BOOL":
		return data.FieldTypeNullableBool
	case "DATETIME", "DATE":
		return data.FieldTypeNullableTime
	case "NULL", "BLOB":
		// SQLite NULL and BLOB types default to STRING
		return data.FieldTypeNullableString
	default:
		// TEXT, VARCHAR, CHAR, and unknown types
		return data.FieldTypeNullableString
	}
}

// convertValueForField converts a raw JSON-decoded value from rqlite
// to the appropriate Go type matching the target Grafana FieldType.
//
// rqlite returns JSON-decoded values:
//   - float64 for all numeric types (JSON numbers)
//   - string for text types
//   - nil for NULL
//   - bool for boolean types
//
// For time columns, rqlite returns Unix timestamps as float64 (seconds).
func convertValueForField(val interface{}, colType string, fieldType data.FieldType) interface{} {
	if val == nil {
		return nil
	}

	switch fieldType {
	case data.FieldTypeNullableInt64:
		return toNullableInt64(val)
	case data.FieldTypeNullableFloat64:
		return toNullableFloat64(val)
	case data.FieldTypeNullableBool:
		return toNullableBool(val)
	case data.FieldTypeNullableTime:
		return toNullableTime(val)
	case data.FieldTypeNullableString:
		return toNullableString(val)
	default:
		return toNullableString(val)
	}
}

// toNullableInt64 converts a value to *int64 for NullableInt64 fields.
func toNullableInt64(val interface{}) *int64 {
	switch v := val.(type) {
	case float64:
		i := int64(v)
		return &i
	case int64:
		return &v
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil
		}
		return &i
	case bool:
		if v {
			i := int64(1)
			return &i
		}
		i := int64(0)
		return &i
	default:
		return nil
	}
}

// toNullableFloat64 converts a value to *float64 for NullableFloat64 fields.
func toNullableFloat64(val interface{}) *float64 {
	switch v := val.(type) {
	case float64:
		return &v
	case int64:
		f := float64(v)
		return &f
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil
		}
		return &f
	default:
		return nil
	}
}

// toNullableBool converts a value to *bool for NullableBool fields.
func toNullableBool(val interface{}) *bool {
	switch v := val.(type) {
	case bool:
		return &v
	case float64:
		b := v != 0
		return &b
	case int64:
		b := v != 0
		return &b
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil
		}
		return &b
	default:
		return nil
	}
}

// Common SQLite datetime formats that may appear in query results.
// SQLite doesn't have a native datetime type, so these are stored as TEXT.
var sqliteTimeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",           // ISO 8601 without timezone
	"2006-01-02 15:04:05.999999999", // SQLite DATETIME with fractional seconds
	"2006-01-02 15:04:05",           // SQLite DATETIME
	"2006-01-02T15:04",              // ISO 8601 without seconds
	"2006-01-02 15:04",              // SQLite DATETIME without seconds
	"2006-01-02",                    // Date only
}

// toNullableTime converts a value to *time.Time for NullableTime fields.
// Supports: Unix timestamps (seconds/milliseconds as float64/int64),
// and common SQLite datetime string formats.
func toNullableTime(val interface{}) *time.Time {
	switch v := val.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case float64:
		// Determine if seconds or milliseconds based on magnitude
		// Unix timestamps in seconds are typically < 4102444800 (year 2100)
		// Unix timestamps in milliseconds are typically > 4102444800000
		var t time.Time
		if v > 1e12 {
			// Milliseconds
			t = time.Unix(0, int64(v)*int64(time.Millisecond))
		} else {
			// Seconds
			sec := int64(v)
			nsec := int64((v - float64(sec)) * 1e9)
			t = time.Unix(sec, nsec)
		}
		return &t
	case int64:
		var t time.Time
		if v > 1e12 {
			t = time.Unix(0, v*int64(time.Millisecond))
		} else {
			t = time.Unix(v, 0)
		}
		return &t
	case string:
		// Try parsing as Unix timestamp
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return toNullableTime(f)
		}
		// Try common SQLite datetime formats
		for _, format := range sqliteTimeFormats {
			if t, err := time.Parse(format, v); err == nil {
				return &t
			}
		}
		return nil
	default:
		return nil
	}
}

// toNullableString converts a value to *string for NullableString fields.
func toNullableString(val interface{}) *string {
	switch v := val.(type) {
	case string:
		return &v
	case float64:
		s := strconv.FormatFloat(v, 'f', -1, 64)
		return &s
	case int64:
		s := strconv.FormatInt(v, 10)
		return &s
	case bool:
		if v {
			s := "1"
			return &s
		}
		s := "0"
		return &s
	default:
		s := fmt.Sprintf("%v", val)
		return &s
	}
}
