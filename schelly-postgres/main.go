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

	sugar.Infof("====Starting Postgres Schelly Backup Repo v.1====")

	postgresBackuper := postgresrepo.PostgresBackuper{}
	err := schellyhook.Initialize(postgresBackuper)

	if err != nil {
		sugar.Errorf("Error initializating Schellyhook. err=%s", err)
		os.Exit(1)
	}

	sugar.Infof("====Postgres Schelly Backup Repo Started====")

}
