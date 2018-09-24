package main

import (
	"os"

	"github.com/flaviostutz/schelly-webhook/schellyhook"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infof("====Starting Postgres Schelly Backup Provider v.1====")

	postgresBackuper := PostgresBackuper{}
	err := schellyhook.Initialize(postgresBackuper)

	if err != nil {
		sugar.Errorf("Error initializating Schellyhook. err=%s", err)
		os.Exit(1)
	}

	sugar.Infof("====Postgres Schelly Backup Provider Started====")

}
