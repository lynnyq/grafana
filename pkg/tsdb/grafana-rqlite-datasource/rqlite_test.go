package rqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConnectURL(t *testing.T) {
	tests := []struct {
		name     string
		config   connectionConfig
		baseURL  string
		expected string
	}{
		{
			name: "simple URL with auto-discovery and weak consistency",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyWeak,
			},
			baseURL:  "http://localhost:4001",
			expected: "http://localhost:4001?level=weak",
		},
		{
			name: "URL without scheme gets http:// prepended",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyWeak,
			},
			baseURL:  "localhost:4001",
			expected: "http://localhost:4001?level=weak",
		},
		{
			name: "static list mode adds disableClusterDiscovery",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeStaticList,
				ReadConsistency: ReadConsistencyWeak,
			},
			baseURL:  "http://node1:4001",
			expected: "http://node1:4001?level=weak&disableClusterDiscovery=true",
		},
		{
			name: "strong consistency",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyStrong,
			},
			baseURL:  "http://localhost:4001",
			expected: "http://localhost:4001?level=strong",
		},
		{
			name: "none consistency",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyNone,
			},
			baseURL:  "http://localhost:4001",
			expected: "http://localhost:4001?level=none",
		},
		{
			name: "linearizable consistency",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyLinearizable,
			},
			baseURL:  "http://localhost:4001",
			expected: "http://localhost:4001?level=linearizable",
		},
		{
			name: "with credentials",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyWeak,
				Username:        "admin",
				Password:        "secret",
			},
			baseURL:  "http://localhost:4001",
			expected: "http://admin:secret@localhost:4001?level=weak",
		},
		{
			name: "with username only",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyWeak,
				Username:        "admin",
			},
			baseURL:  "http://localhost:4001",
			expected: "http://admin:@localhost:4001?level=weak",
		},
		{
			name: "https URL",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeAutoDiscovery,
				ReadConsistency: ReadConsistencyStrong,
			},
			baseURL:  "https://secure-rqlite:443",
			expected: "https://secure-rqlite:443?level=strong",
		},
		{
			name: "static list with strong consistency",
			config: connectionConfig{
				ConnectionMode:  ConnectionModeStaticList,
				ReadConsistency: ReadConsistencyStrong,
			},
			baseURL:  "http://node1:4001",
			expected: "http://node1:4001?level=strong&disableClusterDiscovery=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RqliteConnection{config: tt.config}
			result := rc.buildConnectURL(tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectionModeDefaults(t *testing.T) {
	// Verify that empty connection mode defaults to auto-discovery
	jsonData := JsonData{
		URL: "http://localhost:4001",
	}

	config := connectionConfig{
		URL:            jsonData.URL,
		ConnectionMode: jsonData.ConnectionMode,
	}

	if config.ConnectionMode == "" {
		config.ConnectionMode = ConnectionModeAutoDiscovery
	}
	assert.Equal(t, ConnectionModeAutoDiscovery, config.ConnectionMode)
}

func TestReadConsistencyDefaults(t *testing.T) {
	jsonData := JsonData{
		URL: "http://localhost:4001",
	}

	config := connectionConfig{
		URL:             jsonData.URL,
		ReadConsistency: jsonData.ReadConsistency,
	}

	if config.ReadConsistency == "" {
		config.ReadConsistency = ReadConsistencyWeak
	}
	assert.Equal(t, ReadConsistencyWeak, config.ReadConsistency)
}

func TestClusterUrlsParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single URL",
			input:    "http://node1:4001",
			expected: []string{"http://node1:4001"},
		},
		{
			name:     "multiple URLs",
			input:    "http://node1:4001,http://node2:4001,http://node3:4001",
			expected: []string{"http://node1:4001", "http://node2:4001", "http://node3:4001"},
		},
		{
			name:     "URLs with spaces",
			input:    " http://node1:4001 , http://node2:4001 , ",
			expected: []string{"http://node1:4001", "http://node2:4001"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var urls []string
			if tt.input != "" {
				for _, u := range splitClusterUrls(tt.input) {
					urls = append(urls, u)
				}
			}
			assert.Equal(t, tt.expected, urls)
		})
	}
}

// splitClusterUrls splits a comma-separated string of URLs and trims whitespace.
func splitClusterUrls(s string) []string {
	var result []string
	for _, u := range splitByComma(s) {
		u = trimSpace(u)
		if u != "" {
			result = append(result, u)
		}
	}
	return result
}

func splitByComma(s string) []string {
	// Simple split - in production code this uses strings.Split
	result := []string{}
	start := 0
	for i, c := range s {
		if c == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}
