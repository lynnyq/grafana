package rqlite

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/gtime"
)

// rqliteMacroEngine implements SQLMacroEngine for SQLite/rqlite-specific macros.
type rqliteMacroEngine struct {
	timeInterval string
}

func newRqliteMacroEngine(timeInterval string) *rqliteMacroEngine {
	return &rqliteMacroEngine{timeInterval: timeInterval}
}

// Interpolate replaces rqlite-specific macros in the SQL query.
func (m *rqliteMacroEngine) Interpolate(query *backend.DataQuery, timeRange backend.TimeRange, sql string) (string, error) {
	var err error
	sql, err = m.interpolateTimeGroup(sql, query, timeRange)
	if err != nil {
		return "", err
	}

	sql = m.interpolateTimeFilter(sql, timeRange)
	sql = m.interpolateTimeFilterColumn(sql, timeRange)
	sql = m.interpolateEpoch(sql, timeRange)

	return sql, nil
}

// $__timeGroup(column, interval) → cast((column / intervalSec) * intervalSec as integer)
var timeGroupPattern = regexp.MustCompile(`\$__timeGroup\(([^,]+),\s*([^)]+)\)`)

func (m *rqliteMacroEngine) interpolateTimeGroup(sql string, query *backend.DataQuery, _ backend.TimeRange) (string, error) {
	matches := timeGroupPattern.FindAllStringSubmatchIndex(sql, -1)
	if len(matches) == 0 {
		return sql, nil
	}

	result := sql
	for _, match := range matches {
		column := sql[match[2]:match[3]]
		intervalStr := sql[match[4]:match[5]]

		intervalMs, err := m.parseIntervalToMs(intervalStr, query.Interval.Milliseconds())
		if err != nil {
			return "", fmt.Errorf("failed to parse interval in $__timeGroup: %w", err)
		}

		intervalSec := intervalMs / 1000
		replacement := fmt.Sprintf("cast((%s / %d) * %d as integer)", column, intervalSec, intervalSec)
		result = strings.Replace(result, sql[match[0]:match[1]], replacement, 1)
	}

	return result, nil
}

// $__timeFilter(column) → column >= from AND column <= to
var timeFilterPattern = regexp.MustCompile(`\$__timeFilter\(([^)]+)\)`)

func (m *rqliteMacroEngine) interpolateTimeFilter(sql string, timeRange backend.TimeRange) string {
	return timeFilterPattern.ReplaceAllStringFunc(sql, func(match string) string {
		submatch := timeFilterPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		column := submatch[1]
		from := timeRange.From.UTC().Unix()
		to := timeRange.To.UTC().Unix()
		return fmt.Sprintf("%s >= %d AND %s <= %d", column, from, column, to)
	})
}

// $__timeFilterColumn(column) → same as $__timeFilter but for explicit column references
var timeFilterColumnPattern = regexp.MustCompile(`\$__timeFilterColumn\(([^)]+)\)`)

func (m *rqliteMacroEngine) interpolateTimeFilterColumn(sql string, timeRange backend.TimeRange) string {
	return timeFilterColumnPattern.ReplaceAllStringFunc(sql, func(match string) string {
		submatch := timeFilterColumnPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		column := submatch[1]
		from := timeRange.From.UTC().Unix()
		to := timeRange.To.UTC().Unix()
		return fmt.Sprintf("%s >= %d AND %s <= %d", column, from, column, to)
	})
}

// $__unixEpochFrom() and $__unixEpochTo()
func (m *rqliteMacroEngine) interpolateEpoch(sql string, timeRange backend.TimeRange) string {
	sql = strings.ReplaceAll(sql, "$__unixEpochFrom()", fmt.Sprintf("%d", timeRange.From.UTC().Unix()))
	sql = strings.ReplaceAll(sql, "$__unixEpochTo()", fmt.Sprintf("%d", timeRange.To.UTC().Unix()))
	return sql
}

// parseIntervalToMs converts an interval string (e.g., "1m", "5m", "1h") to milliseconds.
func (m *rqliteMacroEngine) parseIntervalToMs(intervalStr string, defaultMs int64) (int64, error) {
	intervalStr = strings.TrimSpace(intervalStr)

	switch intervalStr {
	case "$__interval":
		return defaultMs, nil
	case "$__interval_ms":
		return defaultMs, nil
	}

	// Try to parse as a Go duration string
	d, err := gtime.ParseInterval(intervalStr)
	if err == nil {
		return d.Milliseconds(), nil
	}

	// Try to parse as a plain number (seconds)
	if sec, err := strconv.ParseInt(intervalStr, 10, 64); err == nil {
		return sec * 1000, nil
	}

	return defaultMs, nil
}
