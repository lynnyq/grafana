package rqlite

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestInterpolateGlobal(t *testing.T) {
	query := backend.DataQuery{
		Interval: 5 * time.Minute,
		TimeRange: backend.TimeRange{
			From: time.Unix(1718000000, 0),
			To:   time.Unix(1718003600, 0),
		},
	}
	timeRange := query.TimeRange

	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "$__interval_ms",
			sql:      "SELECT * FROM t WHERE time > $__interval_ms",
			expected: "SELECT * FROM t WHERE time > 300000",
		},
		{
			name:     "$__interval",
			sql:      "SELECT * FROM t WHERE time > $__interval",
			expected: "SELECT * FROM t WHERE time > 5m",
		},
		{
			name:     "$__unixEpochFrom",
			sql:      "SELECT * FROM t WHERE ts >= $__unixEpochFrom()",
			expected: "SELECT * FROM t WHERE ts >= 1718000000",
		},
		{
			name:     "$__unixEpochTo",
			sql:      "SELECT * FROM t WHERE ts <= $__unixEpochTo()",
			expected: "SELECT * FROM t WHERE ts <= 1718003600",
		},
		{
			name:     "$__from milliseconds",
			sql:      "SELECT * FROM t WHERE time >= $__from",
			expected: "SELECT * FROM t WHERE time >= 1718000000000",
		},
		{
			name:     "$__to milliseconds",
			sql:      "SELECT * FROM t WHERE time <= $__to",
			expected: "SELECT * FROM t WHERE time <= 1718003600000",
		},
		{
			name:     "no macros",
			sql:      "SELECT * FROM t",
			expected: "SELECT * FROM t",
		},
		{
			name:     "multiple macros",
			sql:      "SELECT * FROM t WHERE ts >= $__unixEpochFrom() AND ts <= $__unixEpochTo() AND period > $__interval",
			expected: "SELECT * FROM t WHERE ts >= 1718000000 AND ts <= 1718003600 AND period > 5m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateGlobal(query, timeRange, "1m", tt.sql)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryJSON_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected queryJSON
	}{
		{
			name:     "rawSql and format",
			json:     `{"rawSql": "SELECT * FROM t", "format": "time_series"}`,
			expected: queryJSON{RawSql: "SELECT * FROM t", Format: "time_series"},
		},
		{
			name:     "rawSql only",
			json:     `{"rawSql": "SELECT 1"}`,
			expected: queryJSON{RawSql: "SELECT 1", Format: ""},
		},
		{
			name:     "empty json",
			json:     `{}`,
			expected: queryJSON{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var qj queryJSON
			err := json.Unmarshal([]byte(tt.json), &qj)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, qj)
		})
	}
}

func TestApplyTimeSeriesFormat_IntegerTimeColumn(t *testing.T) {
	// Simulates the case where time is stored as NullableTime (auto-detected by column name)
	// and the user requests time_series format
	handler := &rqliteHandler{
		rowLimit: defaultRowLimit,
	}

	t1 := time.Unix(1718000000, 0)
	t2 := time.Unix(1718000060, 0)

	frame := data.NewFrame("test",
		data.NewField("time", nil, []*time.Time{&t1, &t2}),
		data.NewField("value", nil, []*float64{float64Ptr(85.5), float64Ptr(90.2)}),
	)

	frames := handler.applyTimeSeriesFormat(frame)
	require.NotEmpty(t, frames)

	// Should produce individual frames per metric
	assert.Len(t, frames, 1)
	timeField := frames[0].Fields[0]
	assert.Equal(t, data.FieldTypeNullableTime, timeField.Type())
	valueField := frames[0].Fields[1]
	assert.Equal(t, "value", valueField.Name)
}

func TestApplyTimeSeriesFormat_NoTimeColumn(t *testing.T) {
	handler := &rqliteHandler{
		rowLimit: defaultRowLimit,
	}

	frame := data.NewFrame("test",
		data.NewField("name", nil, []*string{strPtr("Alice")}),
		data.NewField("value", nil, []*float64{float64Ptr(85.5)}),
	)

	frames := handler.applyTimeSeriesFormat(frame)
	require.NotEmpty(t, frames)

	// Should have a warning notice about missing time column
	require.NotNil(t, frames[0].Meta)
	assert.Len(t, frames[0].Meta.Notices, 1)
	assert.Equal(t, data.NoticeSeverityWarning, frames[0].Meta.Notices[0].Severity)
}

func TestApplyTimeSeriesFormat_AlreadyTimeType(t *testing.T) {
	handler := &rqliteHandler{
		rowLimit: defaultRowLimit,
	}

	t1 := time.Unix(1718000000, 0)
	t2 := time.Unix(1718000060, 0)

	frame := data.NewFrame("test",
		data.NewField("time", nil, []*time.Time{&t1, &t2}),
		data.NewField("value", nil, []*float64{float64Ptr(85.5), float64Ptr(90.2)}),
	)

	frames := handler.applyTimeSeriesFormat(frame)
	require.NotEmpty(t, frames)

	// Time field should remain NullableTime
	timeField := frames[0].Fields[0]
	assert.Equal(t, data.FieldTypeNullableTime, timeField.Type())
}

func TestExecuteQuery_EmptyRawSql(t *testing.T) {
	handler := &rqliteHandler{
		rowLimit: defaultRowLimit,
	}

	query := backend.DataQuery{
		RefID: "A",
		JSON:  []byte(`{"rawSql": ""}`),
	}

	dr := handler.executeQuery(nil, query)
	assert.NoError(t, dr.Error)
	assert.Empty(t, dr.Frames)
}

func TestExecuteQuery_InvalidJSON(t *testing.T) {
	handler := &rqliteHandler{
		rowLimit: defaultRowLimit,
	}

	query := backend.DataQuery{
		RefID: "A",
		JSON:  []byte(`invalid json`),
	}

	dr := handler.executeQuery(nil, query)
	assert.Error(t, dr.Error)
}
