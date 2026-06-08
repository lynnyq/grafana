package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	rqlite "github.com/grafana/grafana/pkg/tsdb/grafana-rqlite-datasource"
)

func main() {
	logger := backend.NewLoggerWith()
	if err := datasource.Manage("grafana-rqlite-datasource", rqlite.NewInstanceSettings(logger), datasource.ManageOpts{}); err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}
