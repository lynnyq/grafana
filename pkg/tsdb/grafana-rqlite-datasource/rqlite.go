package rqlite

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/rqlite/gorqlite"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// RqliteConnection manages a gorqlite connection with cluster support.
type RqliteConnection struct {
	mu     sync.RWMutex
	conn   *gorqlite.Connection
	config connectionConfig
	log    log.Logger
}

type connectionConfig struct {
	URL             string
	ClusterUrls     []string
	ConnectionMode  ConnectionMode
	ReadConsistency ReadConsistency
	Username        string
	Password        string
	TLSSkipVerify   bool
}

// newRqliteConnection creates a new gorqlite connection based on the provided configuration.
func newRqliteConnection(_ context.Context, jsonData JsonData, decryptedSecureJsonData map[string]string, logger log.Logger) (*RqliteConnection, error) {
	config := connectionConfig{
		URL:             jsonData.URL,
		ConnectionMode:  jsonData.ConnectionMode,
		ReadConsistency: jsonData.ReadConsistency,
		Username:        jsonData.Username,
		Password:        decryptedSecureJsonData["password"],
		TLSSkipVerify:   jsonData.TLSSkipVerify,
	}

	if jsonData.ClusterUrls != "" {
		for _, u := range strings.Split(jsonData.ClusterUrls, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				config.ClusterUrls = append(config.ClusterUrls, u)
			}
		}
	}

	if config.ConnectionMode == "" {
		config.ConnectionMode = ConnectionModeAutoDiscovery
	}

	if config.ReadConsistency == "" {
		config.ReadConsistency = ReadConsistencyWeak
	}

	rc := &RqliteConnection{
		config: config,
		log:    logger,
	}

	conn, err := rc.dial()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rqlite: %w", err)
	}

	rc.conn = conn
	return rc, nil
}

// dial establishes a gorqlite connection based on the connection mode.
func (rc *RqliteConnection) dial() (*gorqlite.Connection, error) {
	var connectURL string

	switch rc.config.ConnectionMode {
	case ConnectionModeStaticList:
		if len(rc.config.ClusterUrls) > 0 {
			connectURL = rc.config.ClusterUrls[0]
		} else {
			return nil, fmt.Errorf("no cluster URLs provided for static list mode")
		}
	case ConnectionModeAutoDiscovery:
		fallthrough
	default:
		if rc.config.URL == "" {
			return nil, fmt.Errorf("no URL provided for auto-discovery mode")
		}
		connectURL = rc.config.URL
	}

	// Build connection URL with credentials and options
	connectURL = rc.buildConnectURL(connectURL)

	// Create HTTP client with TLS config
	httpClient := &http.Client{}
	if rc.config.TLSSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	conn, err := gorqlite.OpenWithClient(connectURL, httpClient)
	if err != nil {
		return nil, fmt.Errorf("gorqlite open failed: %w", err)
	}

	// Set consistency level
	rc.setConsistency(conn)

	return conn, nil
}

// buildConnectURL constructs the gorqlite connection URL with credentials and query parameters.
func (rc *RqliteConnection) buildConnectURL(baseURL string) string {
	// Ensure URL has scheme
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	// Insert credentials if provided
	if rc.config.Username != "" || rc.config.Password != "" {
		schemeEnd := strings.Index(baseURL, "://")
		scheme := baseURL[:schemeEnd+3]
		rest := baseURL[schemeEnd+3:]
		baseURL = scheme + rc.config.Username + ":" + rc.config.Password + "@" + rest
	}

	// Add query parameters
	params := []string{}
	switch rc.config.ReadConsistency {
	case ReadConsistencyNone:
		params = append(params, "level=none")
	case ReadConsistencyWeak:
		params = append(params, "level=weak")
	case ReadConsistencyStrong:
		params = append(params, "level=strong")
	case ReadConsistencyLinearizable:
		params = append(params, "level=linearizable")
	}

	if rc.config.ConnectionMode == ConnectionModeStaticList {
		params = append(params, "disableClusterDiscovery=true")
	}

	if len(params) > 0 {
		if strings.Contains(baseURL, "?") {
			baseURL += "&" + strings.Join(params, "&")
		} else {
			baseURL += "?" + strings.Join(params, "&")
		}
	}

	return baseURL
}

// setConsistency configures the gorqlite connection consistency level.
func (rc *RqliteConnection) setConsistency(conn *gorqlite.Connection) {
	switch rc.config.ReadConsistency {
	case ReadConsistencyNone:
		conn.SetConsistencyLevel(gorqlite.ConsistencyLevelNone)
	case ReadConsistencyWeak:
		conn.SetConsistencyLevel(gorqlite.ConsistencyLevelWeak)
	case ReadConsistencyStrong:
		conn.SetConsistencyLevel(gorqlite.ConsistencyLevelStrong)
	case ReadConsistencyLinearizable:
		conn.SetConsistencyLevel(gorqlite.ConsistencyLevelLinearizable)
	default:
		conn.SetConsistencyLevel(gorqlite.ConsistencyLevelWeak)
	}
}

// Query executes a SQL query against the rqlite cluster.
func (rc *RqliteConnection) Query(ctx context.Context, sql string) (*gorqlite.QueryResult, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if rc.conn == nil {
		return nil, fmt.Errorf("rqlite connection is not initialized")
	}

	rows, err := rc.conn.QueryOneContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("rqlite query failed: %w", err)
	}

	return &rows, nil
}

// CheckHealth verifies connectivity to the rqlite cluster.
func (rc *RqliteConnection) CheckHealth(ctx context.Context) (*backend.CheckHealthResult, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if rc.conn == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "rqlite connection is not initialized",
		}, nil
	}

	leader, err := rc.conn.Leader()
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("failed to get rqlite leader: %v", err),
		}, nil
	}

	peers, err := rc.conn.Peers()
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("failed to get rqlite peers: %v", err),
		}, nil
	}

	// Verify query capability by executing a simple query
	rows, err := rc.conn.QueryOneContext(ctx, "SELECT 1")
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("failed to execute test query: %v", err),
		}, nil
	}

	_ = rows

	return &backend.CheckHealthResult{
		Status: backend.HealthStatusOk,
		Message: fmt.Sprintf("rqlite cluster is healthy. Leader: %s, Peers: %d",
			leader, len(peers)),
	}, nil
}

// Close closes the gorqlite connection.
func (rc *RqliteConnection) Close() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.conn != nil {
		rc.conn.Close()
		rc.conn = nil
	}
	return nil
}
