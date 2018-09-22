package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/flaviostutz/schelly-webhook/schellyhook"
	"go.uber.org/zap"
)

// backups directory where the backup files will be placed
var backupsDir *string

// General options:
var fileName *string //output file or directory name
var splitFile *bool  //output file or directory name

// Options controlling the output content:
var dataOnly *bool   // dump only the data, not the schema
var schemaOnly *bool // dump only the schema, no data
var encoding *string // dump the data in encoding ENCODING
// var schema *string[]           // dump the named schema(s) only
// var excludeSchema *string[]    // do NOT dump the named schema(s)
// var table *string[]            // dump the named table(s) only
// var excludeTable *string[]     // do NOT dump the named table(s)
// var excludeTableData *string[] // do NOT dump data for the named table(s)

// Connection options:
var dbname *string   // database to dump
var host *string     // database server host or socket directory
var port *int        // database server port number
var username *string // connect as specified database user
var password *string // force password prompt (should happen automatically)

//PostgresBackuper sample backuper
type PostgresBackuper struct{}

//Init check the pg_dump version
func (sb PostgresBackuper) Init() error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	info, err := schellyhook.ExecShell("pg_dump --version")
	if err != nil {
		sugar.Errorf("Couldn't retrieve pg_dump version. err=%s", err)
		return err
	}

	if *backupsDir == "" {
		return fmt.Errorf("backup-dir arg must be defined")
	}
	if strings.Contains(*fileName, "--") {
		return fmt.Errorf("Cannot use `--` on file name. Please change the filename and try again; you can still use `-`")
	}
	if *host == "" {
		return fmt.Errorf("`database host` (--host) arg must be set. It can be an IP address or a domain name")
	}
	if *port <= 0 {
		return fmt.Errorf("`database port` (--port) arg must be a valid value, such as 5432")
	}
	if *dbname == "" {
		return fmt.Errorf("`dbname` (--dbname) arg must be set")
	}
	if *username == "" {
		return fmt.Errorf("`username` (--username) arg must be set")
	}
	if *password == "" {
		return fmt.Errorf("`password` (--password) arg must be set")
	}

	pgPassStringBytes := []byte(*host + ":" + strconv.Itoa(*port) + ":" + *username + ":" + *dbname + ":" + *password)

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(usr.HomeDir+".pgpass", pgPassStringBytes, 0644)
	if err != nil {
		sugar.Errorf("Error writing .pgpass file. err: %s", err)
		return err
	}

	pgPassFile, err := ioutil.ReadFile(usr.HomeDir + ".pgpass")
	if err != nil {
		sugar.Errorf("Error reading .pgpass file. err: %s", err)
		return err
	}
	sugar.Debugf("~/.pgpass file created. Contents: %s", pgPassFile)

	sugar.Infof("Postgres Repository ready to work. Version: ", info)

	return nil
}

//RegisterFlags register command line flags
func (sb PostgresBackuper) RegisterFlags() error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	// General options:
	backupsDir = flag.String("backup-dir", "/var/backups/database", "--backup-dir=FILENAME -> output file path and name")
	fileName = flag.String("file-name", "database_dump", "--file-name=FILENAME -> output file path and name")
	splitFile = flag.Bool("split-file", false, "--split-file -> split the backup on multiple files on a directory (pg_dump --format=d)")

	// Options controlling the output content:
	dataOnly = flag.Bool("data-only", false, "--data-only -> dump only the data, not the schema")
	schemaOnly = flag.Bool("schema-only", false, "--schema-only -> dump only the schema, no data")
	encoding = flag.String("encoding", "UTF-8", "--encoding=ENCODING -> dump the data in encoding ENCODING")
	// schema = flag.Var("schema","","--schema=SCHEMA -> dump the named schema(s) only")
	// excludeSchema = flag.Var("exclude-schema", "", "--exclude-schema=SCHEMA -> do NOT dump the named schema(s)")
	// table = flag.Var("table", "", "--table=TABLE -> dump the named table(s) only")
	// excludeTable = flag.Var("exclude-table", "", "--exclude-table=TABLE -> do NOT dump the named table(s)")
	// excludeTableData = flag.Var("exclude-table-data", "", "--exclude-table-data=TABLE -> do NOT dump data for the named table(s)")

	// Connection options:
	dbname = flag.String("dbname", "", "--dbname=DBNAME -> database to dump")
	host = flag.String("host", "", "--host=HOSTNAME -> database server host or socket directory")
	port = flag.Int("port", 5432, "--port=PORT -> database server port number")
	username = flag.String("username", "postgres", "--username=NAME -> connect as specified database user")
	password = flag.String("password", "", " --password -> password to be placed on ~/.pgpass")

	// flag.Parse() //invoked by the hook
	sugar.Infof("Flags registration completed")

	return nil
}

//CreateNewBackup creates a new backup
func (sb PostgresBackuper) CreateNewBackup(apiID string, timeout time.Duration, shellContext *schellyhook.ShellContext) error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infof("CreateNewBackup() apiID=%s timeout=%d s", apiID, timeout.Seconds)
	sugar.Infof("Running Postgres pg_dump backup")

	pgDumpID := time.Now().Format("20060102150405")
	fileString := "--file=" + resolveFileName(pgDumpID, apiID)

	dataOnlyString := ""
	if *dataOnly == true {
		dataOnlyString = "--data-only"
	}
	schemaOnlyString := ""
	if *schemaOnly == true {
		schemaOnlyString = "--schema-only"
	}
	encodingString := ""
	if encoding != nil {
		encodingString = "--encoding=" + *encoding
	}
	backupFormat := "d"
	if *splitFile == false {
		backupFormat = "p"
	}

	out, err := schellyhook.ExecShellTimeout("pg_dump --verbose --format="+backupFormat+" --jobs=1 --compress=9 --column-inserts --inserts --quote-all-identifiers --clean --create "+fileString+" "+dataOnlyString+" "+schemaOnlyString+" "+encodingString, timeout, shellContext)

	if err != nil {
		status := (*shellContext).CmdRef.Status()
		if status.Exit == -1 {
			sugar.Warnf("PostgresRepo pg_dump command timeout enforced (%d seconds)", (status.StopTs-status.StartTs)/1000000000)
		}
		sugar.Debugf("PostgresRepo pg_dump error. out=%s; err=%s", out, err.Error())
		return err
	}

	sugar.Debugf("PostgresRepo pg_dump backup completed. Output log:")
	sugar.Debugf(out)

	sugar.Infof("Backup success")

	saveDataID(apiID, pgDumpID)

	sugar.Infof("Postgres backup finished")
	return nil
}

func resolveFileName(apiID string, pgDumpID string) string {
	return *backupsDir + "/" + *fileName + "--" + pgDumpID + "---" + apiID
}

//GetAllBackups returns all backups from underlaying backuper. optional for Schelly
func (sb PostgresBackuper) GetAllBackups() ([]schellyhook.SchellyResponse, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("GetAllBackups")
	result, err := schellyhook.ExecShell("ls -l" + *backupsDir)
	if err != nil {
		return nil, err
	}

	backups := make([]schellyhook.SchellyResponse, 0)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		cols := strings.Split(line, " ")
		if i == 0 || len(cols) < 8 {
			continue
		}

		dataID := strings.Split(cols[8], "--")[1]
		sizeMB, err := strconv.ParseFloat(cols[4], 64)
		if err != nil {
			return nil, err
		}
		sizeMB = sizeMB / 1000000.0
		status := "available"

		sr := schellyhook.SchellyResponse{
			DataID:  dataID,
			Status:  status,
			Message: line,
			SizeMB:  sizeMB,
		}
		backups = append(backups, sr)
	}

	return backups, nil
}

//GetBackup get an specific backup along with status
func (sb PostgresBackuper) GetBackup(apiID string) (*schellyhook.SchellyResponse, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("GetBackup apiID=%s", apiID)

	pgDumpID, err0 := getDataID(apiID)
	if err0 != nil {
		sugar.Debugf("pgDumpID not found for apiId %s. err=%s", apiID, err0)
		return nil, nil
	}

	res, err := findBackup(apiID, pgDumpID)
	if err != nil {
		return nil, nil
	}

	return res, nil
}

//DeleteBackup removes current backup from underlaying backup storage
func (sb PostgresBackuper) DeleteBackup(apiID string) error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("DeleteBackup apiID=%s", apiID)

	pgDumpID, err0 := getDataID(apiID)
	if err0 != nil {
		sugar.Debugf("pgDumpID not found for apiId %s. err=%s", apiID, err0)
		return err0
	}

	_, err0 = findBackup(apiID, pgDumpID)
	if err0 != nil {
		sugar.Debugf("Backup apiID %s, pgDumpID %s not found for removal", apiID, pgDumpID)
		return err0
	}

	sugar.Debugf("Backup apiID=%s pgDumpID=%s found. Proceeding to deletion", apiID, pgDumpID)
	result, err := schellyhook.ExecShell("rm " + resolveFileName(apiID, pgDumpID))
	if err != nil {
		if strings.Contains(err.Error(), "100") {
			return fmt.Errorf("Cannot delete this backup because it is too young. Configure $PROTECT_YOUNG_BACKUP_DAYS if needed. err=%s", err)
		} else {
			return err
		}
	}
	sugar.Debugf("result: %s", result)

	sugar.Debugf("Delete apiID %s pgDumpID %s successful", apiID, pgDumpID)
	return nil
}

func findBackup(apiID string, pgDumpID string) (*schellyhook.SchellyResponse, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	result, err := schellyhook.ExecShell("ls -l " + resolveFileName(apiID, pgDumpID))
	if err != nil {
		return nil, err
	}

	cols := strings.Split(result, " ")
	if len(cols) < 6 {
		return nil, fmt.Errorf("pgDumpID file not found or corrupted")
	}
	sugar.Debugf("pgDumpID found. Details: %s", cols)

	size, err := strconv.ParseFloat(cols[4], 64)
	if err != nil {
		return nil, err
	}
	status := "available"

	return &schellyhook.SchellyResponse{
		ID:     apiID,
		DataID: pgDumpID,
		Status: status,
		SizeMB: size / 1000000.0,
	}, nil
}

func getDataID(apiID string) (string, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	result, err := schellyhook.ExecShell("ls -m " + *backupsDir)
	if err == nil {
		return "", err
	}
	files := strings.Split(result, ",")
	for _, file := range files {
		if strings.Contains(file, apiID) {
			if _, err := os.Stat(file); err == nil {
				sugar.Debugf("Found api id reference for %s", apiID)
				_, err0 := ioutil.ReadFile(file)
				if err0 != nil {
					return "", err0
				}
				pgDumpID := strings.Split(file, "--")[1]
				sugar.Debugf("apiID %s <-> pgDumpID %s", apiID, pgDumpID)
				return pgDumpID, nil
			}
		}
	}
	return "", fmt.Errorf("pgDumpID for %s not found", apiID)
}

func saveDataID(apiID string, pgDumpID string) error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("IDs already saved apiID %s <-> pgDumpID %s", apiID, pgDumpID)
	return nil
}
