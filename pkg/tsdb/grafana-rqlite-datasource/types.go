package rqlite

// ConnectionMode determines how the plugin connects to the rqlite cluster.
type ConnectionMode string

const (
	ConnectionModeAutoDiscovery ConnectionMode = "auto-discovery"
	ConnectionModeStaticList    ConnectionMode = "static-list"
)

// ReadConsistency determines the consistency level for read queries.
type ReadConsistency string

const (
	ReadConsistencyNone         ReadConsistency = "none"
	ReadConsistencyWeak         ReadConsistency = "weak"
	ReadConsistencyStrong       ReadConsistency = "strong"
	ReadConsistencyLinearizable ReadConsistency = "linearizable"
)

// JsonData represents the datasource configuration stored in Grafana's database.
type JsonData struct {
	URL              string         `json:"url"`
	ClusterUrls      string         `json:"clusterUrls"`
	ConnectionMode   ConnectionMode `json:"connectionMode"`
	ReadConsistency  ReadConsistency `json:"readConsistency"`
	Username         string         `json:"username"`
	TLSSkipVerify    bool           `json:"tlsSkipVerify"`
	TimeInterval     string         `json:"timeInterval"`
	MaxOpenConns     int            `json:"maxOpenConns"`
	ConnMaxLifetime  int            `json:"connMaxLifetime"`
}

// SecureJsonData holds sensitive configuration data.
type SecureJsonData struct {
	Password string `json:"password"`
}
