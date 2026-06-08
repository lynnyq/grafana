# rqlite Data Source

Data source plugin for [rqlite](https://rqlite.io/), the distributed relational database built on SQLite.

## Features

- Connect to rqlite clusters with auto-discovery or static node list
- SQL query editor with table/column auto-completion
- Builder mode for visual query construction
- Support for time series and table formats
- Multiple read consistency levels (None, Weak, Strong, Linearizable)
- TLS support with optional skip verify

## Configuration

| Option | Description |
|--------|-------------|
| URL | rqlite node URL (e.g., `http://localhost:4001`) |
| Connection Mode | Auto-discovery or Static list |
| Cluster URLs | Comma-separated URLs for static list mode |
| Read Consistency | None, Weak, Strong, or Linearizable |
| Username | Basic auth username |
| Password | Basic auth password |
| TLS Skip Verify | Skip TLS certificate verification |
