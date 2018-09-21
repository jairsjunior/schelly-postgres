package main

import (
	"flag"
	"os"

	"./postgresrepo"
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
	err := postgresBackuper.RegisterFlags()

	if err != nil {
		sugar.Errorf("Error initializating Schellyhook. err=%s", err)
		os.Exit(1)
	}

	flag.Parse()

	sugar.Infof("====Postgres Schelly Backup Repo Started====")

}
