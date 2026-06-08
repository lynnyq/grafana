package rqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestInterpolate_TimeGroup(t *testing.T) {
	engine := newRqliteMacroEngine("1m")
	query := &backend.DataQuery{
		Interval: time.Minute,
	}
	timeRange := backend.TimeRange{
		From: time.Unix(1718000000, 0),
		To:   time.Unix(1718003600, 0),
	}

	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "$__timeGroup with 60s interval",
			sql:      "SELECT $__timeGroup(time, 60s) AS t, avg(value) FROM metrics GROUP BY t",
			expected: "SELECT cast((time / 60) * 60 as integer) AS t, avg(value) FROM metrics GROUP BY t",
		},
		{
			name:     "$__timeGroup with 5m interval",
			sql:      "SELECT $__timeGroup(time, 5m) AS t FROM metrics GROUP BY t",
			expected: "SELECT cast((time / 300) * 300 as integer) AS t FROM metrics GROUP BY t",
		},
		{
			name:     "$__timeGroup with 1h interval",
			sql:      "SELECT $__timeGroup(ts, 1h) AS t FROM metrics GROUP BY t",
			expected: "SELECT cast((ts / 3600) * 3600 as integer) AS t FROM metrics GROUP BY t",
		},
		{
			name:     "$__timeGroup with $__interval",
			sql:      "SELECT $__timeGroup(time, $__interval) AS t FROM metrics GROUP BY t",
			expected: "SELECT cast((time / 60) * 60 as integer) AS t FROM metrics GROUP BY t",
		},
		{
			name:     "no macro",
			sql:      "SELECT * FROM metrics",
			expected: "SELECT * FROM metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Interpolate(query, timeRange, tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInterpolate_TimeFilter(t *testing.T) {
	engine := newRqliteMacroEngine("1m")
	query := &backend.DataQuery{Interval: time.Minute}
	timeRange := backend.TimeRange{
		From: time.Unix(1718000000, 0),
		To:   time.Unix(1718003600, 0),
	}

	tests := []struct {
		name     string
		sql      string
		contains string
	}{
		{
			name:     "$__timeFilter",
			sql:      "SELECT * FROM metrics WHERE $__timeFilter(time)",
			contains: "time >= 1718000000 AND time <= 1718003600",
		},
		{
			name:     "$__timeFilterColumn",
			sql:      "SELECT * FROM metrics WHERE $__timeFilterColumn(created_at)",
			contains: "created_at >= 1718000000 AND created_at <= 1718003600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Interpolate(query, timeRange, tt.sql)
			require.NoError(t, err)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestInterpolate_EpochMacros(t *testing.T) {
	engine := newRqliteMacroEngine("1m")
	query := &backend.DataQuery{Interval: time.Minute}
	timeRange := backend.TimeRange{
		From: time.Unix(1718000000, 0),
		To:   time.Unix(1718003600, 0),
	}

	tests := []struct {
		name     string
		sql      string
		contains string
	}{
		{
			name:     "$__unixEpochFrom",
			sql:      "SELECT * FROM metrics WHERE time >= $__unixEpochFrom()",
			contains: "1718000000",
		},
		{
			name:     "$__unixEpochTo",
			sql:      "SELECT * FROM metrics WHERE time <= $__unixEpochTo()",
			contains: "1718003600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Interpolate(query, timeRange, tt.sql)
			require.NoError(t, err)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestInterpolate_CombinedMacros(t *testing.T) {
	engine := newRqliteMacroEngine("5m")
	query := &backend.DataQuery{Interval: 5 * time.Minute}
	timeRange := backend.TimeRange{
		From: time.Unix(1718000000, 0),
		To:   time.Unix(1718003600, 0),
	}

	sql := `SELECT $__timeGroup(time, $__interval) AS t, avg(value) FROM metrics WHERE $__timeFilter(time) GROUP BY t`
	result, err := engine.Interpolate(query, timeRange, sql)
	require.NoError(t, err)

	// Should contain time group
	assert.Contains(t, result, "cast((time / 300) * 300 as integer)")
	// Should contain time filter
	assert.Contains(t, result, "time >= 1718000000 AND time <= 1718003600")
}
