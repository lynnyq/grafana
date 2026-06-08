package rqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/gtime"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const (
	defaultRowLimit = 1000000
)

// Service implements backend.QueryDataHandler and backend.CheckHealthHandler
// for the rqlite datasource.
type Service struct {
	im instancemgmt.InstanceManager
}

// ProvideService creates a new rqlite Service instance.
func ProvideService() *Service {
	logger := backend.NewLoggerWith("logger", "tsdb.rqlite")
	s := &Service{
		im: datasource.NewInstanceManager(NewInstanceSettings(logger)),
	}
	return s
}

// QueryData handles multiple queries for the rqlite datasource.
func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	handler, err := s.getHandler(ctx, req.PluginContext)
	if err != nil {
		return nil, err
	}
	return handler.QueryData(ctx, req)
}

// CheckHealth handles health checks for the rqlite datasource.
func (s *Service) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	handler, err := s.getHandler(ctx, req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("failed to get rqlite handler: %v", err),
		}, nil
	}
	return handler.CheckHealth(ctx, req)
}

func (s *Service) getHandler(ctx context.Context, pluginCtx backend.PluginContext) (*rqliteHandler, error) {
	i, err := s.im.Get(ctx, pluginCtx)
	if err != nil {
		return nil, err
	}
	return i.(*rqliteHandler), nil
}

// rqliteHandler handles queries and health checks for a specific datasource instance.
type rqliteHandler struct {
	connection   *RqliteConnection
	macroEngine  *rqliteMacroEngine
	log          log.Logger
	rowLimit     int64
	timeInterval string
}

// queryJSON represents the JSON structure of a query.
type queryJSON struct {
	RawSql string `json:"rawSql"`
	Format string `json:"format"`
}

// NewInstanceSettings returns a factory function for creating rqlite handler instances.
func NewInstanceSettings(logger log.Logger) datasource.InstanceFactoryFunc {
	return func(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		jsonData := JsonData{
			ConnectionMode:  ConnectionModeAutoDiscovery,
			ReadConsistency: ReadConsistencyWeak,
		}

		if err := json.Unmarshal(settings.JSONData, &jsonData); err != nil {
			return nil, fmt.Errorf("error reading rqlite settings: %w", err)
		}

		// Apply defaults
		if jsonData.ConnectionMode == "" {
			jsonData.ConnectionMode = ConnectionModeAutoDiscovery
		}
		if jsonData.ReadConsistency == "" {
			jsonData.ReadConsistency = ReadConsistencyWeak
		}

		// Override URL from settings if not set in jsonData
		if jsonData.URL == "" && settings.URL != "" {
			jsonData.URL = settings.URL
		}

		conn, err := newRqliteConnection(ctx, jsonData, settings.DecryptedSecureJSONData, logger)
		if err != nil {
			logger.Error("Failed to connect to rqlite", "error", err)
			return nil, fmt.Errorf("failed to connect to rqlite: %w", err)
		}

		logger.Debug("Successfully connected to rqlite")

		return &rqliteHandler{
			connection:   conn,
			macroEngine:  newRqliteMacroEngine(jsonData.TimeInterval),
			log:          logger,
			rowLimit:     defaultRowLimit,
			timeInterval: jsonData.TimeInterval,
		}, nil
	}
}

// Dispose cleans up the rqlite handler.
func (h *rqliteHandler) Dispose() {
	if h.connection != nil {
		h.connection.Close()
	}
}

// QueryData processes query requests.
func (h *rqliteHandler) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	result := backend.NewQueryDataResponse()
	ch := make(chan *backend.DataResponse, len(req.Queries))
	var wg sync.WaitGroup

	for _, q := range req.Queries {
		wg.Add(1)
		go func(query backend.DataQuery) {
			defer wg.Done()
			dr := h.executeQuery(ctx, query)
			ch <- &dr
		}(q)
	}

	wg.Wait()
	close(ch)

	for _, q := range req.Queries {
		dr := <-ch
		result.Responses[q.RefID] = *dr
	}

	return result, nil
}

// executeQuery processes a single query.
func (h *rqliteHandler) executeQuery(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	var qj queryJSON
	if err := json.Unmarshal(query.JSON, &qj); err != nil {
		return backend.DataResponse{
			Error: fmt.Errorf("error unmarshal query json: %w", err),
		}
	}

	if qj.RawSql == "" {
		return backend.DataResponse{
			Frames: data.Frames{},
		}
	}

	// Apply global macro substitutions
	interpolatedSQL := interpolateGlobal(query, query.TimeRange, h.timeInterval, qj.RawSql)

	// Apply rqlite-specific macro substitutions
	interpolatedSQL, err := h.macroEngine.Interpolate(&query, query.TimeRange, interpolatedSQL)
	if err != nil {
		return backend.DataResponse{
			Error:       fmt.Errorf("macro interpolation failed: %w", err),
			ErrorSource: backend.ErrorSourceDownstream,
		}
	}

	// Execute query via gorqlite
	rows, err := h.connection.Query(ctx, interpolatedSQL)
	if err != nil {
		return backend.DataResponse{
			Error:       fmt.Errorf("rqlite query error: %w", err),
			ErrorSource: backend.ErrorSourceDownstream,
		}
	}

	// Convert to DataFrame
	frame, err := queryResultToFrame(rows, "", h.rowLimit)
	if err != nil {
		return backend.DataResponse{
			Error:       fmt.Errorf("result conversion error: %w", err),
			ErrorSource: backend.ErrorSourcePlugin,
		}
	}

	if frame.Meta == nil {
		frame.Meta = &data.FrameMeta{}
	}
	frame.Meta.ExecutedQueryString = interpolatedSQL

	// Apply time series formatting if requested
	var frames data.Frames
	if qj.Format == "time_series" {
		frames = h.applyTimeSeriesFormat(frame)
	} else {
		frames = data.Frames{frame}
	}

	return backend.DataResponse{
		Frames: frames,
	}
}

// CheckHealth performs a health check against the rqlite cluster.
func (h *rqliteHandler) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return h.connection.CheckHealth(ctx)
}

// applyTimeSeriesFormat converts the frame to time series format.
// Following the pattern from frser-sqlite-datasource:
// 1. Find the time column and convert numeric values to time.Time
// 2. Use LongToWide conversion for long-format time series
// 3. Split wide-format time series into individual frames per metric
func (h *rqliteHandler) applyTimeSeriesFormat(frame *data.Frame) data.Frames {
	// Find time column
	timeIdx := -1
	for i, field := range frame.Fields {
		if field.Type() == data.FieldTypeNullableTime {
			timeIdx = i
			break
		}
	}

	if timeIdx == -1 {
		// No time column found, return as-is with a warning
		if frame.Meta == nil {
			frame.Meta = &data.FrameMeta{}
		}
		frame.AppendNotices(data.Notice{
			Severity: data.NoticeSeverityWarning,
			Text:     "Time series format requested but no time column found. Displaying as table.",
		})
		return data.Frames{frame}
	}

	// If time column is already NullableTime, check if we need LongToWide conversion
	tsSchema := frame.TimeSeriesSchema()

	// If already in wide format, split into individual frames
	if tsSchema.Type == data.TimeSeriesTypeWide {
		return h.splitWideFrame(frame, tsSchema)
	}

	// If in long format, convert to wide first
	if tsSchema.Type == data.TimeSeriesTypeLong {
		wideFrame, err := data.LongToWide(frame, nil)
		if err != nil {
			h.log.Error("Could not convert from long to wide time-series", "error", err)
			return data.Frames{frame}
		}
		executedQuery := ""
		if frame.Meta != nil {
			executedQuery = frame.Meta.ExecutedQueryString
		}
		wideFrame.Meta = &data.FrameMeta{ExecutedQueryString: executedQuery}

		// Remove empty fields that may have been created during conversion
		wideFrame = h.removeNullOnlyFields(wideFrame)

		wideTsSchema := wideFrame.TimeSeriesSchema()
		return h.splitWideFrame(wideFrame, wideTsSchema)
	}

	// No time series structure found, return as-is
	return data.Frames{frame}
}

// splitWideFrame splits a wide-format time series frame into individual frames,
// one per metric field. This follows the frser-sqlite pattern and ensures
// compatibility with Grafana panels that expect individual time series frames.
func (h *rqliteHandler) splitWideFrame(frame *data.Frame, tsSchema data.TimeSeriesSchema) data.Frames {
	if tsSchema.TimeIndex < 0 {
		return data.Frames{frame}
	}

	var frames data.Frames
	executedQuery := ""
	if frame.Meta != nil {
		executedQuery = frame.Meta.ExecutedQueryString
	}
	for idx, field := range frame.Fields {
		if idx == tsSchema.TimeIndex {
			continue
		}
		partialFrame := data.NewFrame("", frame.Fields[tsSchema.TimeIndex], field)
		partialFrame.Meta = &data.FrameMeta{ExecutedQueryString: executedQuery}
		frames = append(frames, partialFrame)
	}

	if len(frames) == 0 {
		return data.Frames{frame}
	}
	return frames
}

// removeNullOnlyFields removes fields from a wide-format frame that contain only null values
// and have an empty "name" label. These can be created during LongToWide conversion when
// there are empty factor values. This follows the frser-sqlite pattern.
func (h *rqliteHandler) removeNullOnlyFields(frame *data.Frame) *data.Frame {
	emptyFieldIndexes := map[int]bool{}
	for idx, field := range frame.Fields {
		if field.Labels["name"] == "" && fieldHasOnlyNulls(field) {
			emptyFieldIndexes[idx] = true
		}
	}
	if len(emptyFieldIndexes) > 0 {
		var filteredFields []*data.Field
		for idx, field := range frame.Fields {
			if !emptyFieldIndexes[idx] {
				filteredFields = append(filteredFields, field)
			}
		}
		frame.Fields = filteredFields
	}
	return frame
}

// fieldHasOnlyNulls checks if a field contains only null values.
func fieldHasOnlyNulls(field *data.Field) bool {
	for row := 0; row < field.Len(); row++ {
		if _, isNil := field.ConcreteAt(row); isNil {
			return false
		}
	}
	return true
}

// interpolateGlobal provides global macro substitutions shared across all SQL datasources.
func interpolateGlobal(query backend.DataQuery, timeRange backend.TimeRange, _ string, sql string) string {
	interval := query.Interval

	sql = strings.ReplaceAll(sql, "$__interval_ms", strconv.FormatInt(interval.Milliseconds(), 10))
	sql = strings.ReplaceAll(sql, "$__interval", gtime.FormatInterval(interval))
	// $__from/$__to in milliseconds (Grafana standard, used by frser-sqlite and other SQL datasources)
	sql = strings.ReplaceAll(sql, "$__from", strconv.FormatInt(timeRange.From.UTC().UnixMilli(), 10))
	sql = strings.ReplaceAll(sql, "$__to", strconv.FormatInt(timeRange.To.UTC().UnixMilli(), 10))
	// $__unixEpochFrom/$__unixEpochTo in seconds (for SQLite strftime/date functions)
	sql = strings.ReplaceAll(sql, "$__unixEpochFrom()", strconv.FormatInt(timeRange.From.UTC().Unix(), 10))
	sql = strings.ReplaceAll(sql, "$__unixEpochTo()", strconv.FormatInt(timeRange.To.UTC().Unix(), 10))

	return sql
}
