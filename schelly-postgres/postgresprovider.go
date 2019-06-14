package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/flaviostutz/schelly-webhook/schellyhook"
	"go.uber.org/zap"
)

var dataStringSeparator string

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

// Azure options:
var azureStorage *bool    // azure storage active
var accountName *string   // azure account name
var accountKey *string    // azure account key
var containerName *string // azure container name

//PostgresBackuper sample backuper
type PostgresBackuper struct{}

//Init check the pg_dump version
func (sb PostgresBackuper) Init() error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	dataStringSeparator = "---"

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
	basicDir := "/var/backups"
	err = mkDirs(basicDir)
	if err != nil {
		return fmt.Errorf("Error creating basic workdir /var/backups")
	}

	// About .pgpass file
	// https://www.postgresql.org/docs/9.3/static/libpq-pgpass.html
	pgPassFilePath := basicDir + "/.pgpass"
	os.Setenv("PGPASSFILE", pgPassFilePath)
	pgPassStringBytes := []byte("*:*:*:*:" + *password)
	err = ioutil.WriteFile(pgPassFilePath, pgPassStringBytes, 0600)
	if err != nil {
		sugar.Errorf("Error writing .pgpass file. err: %s", err)
		return err
	}

	pgPassFile, err := ioutil.ReadFile(pgPassFilePath)
	if err != nil {
		sugar.Errorf("Error reading .pgpass file. err: %s", err)
		return err
	}
	sugar.Debugf(pgPassFilePath+" file created. Contents: %s", pgPassFile)

	err = mkDirs(*backupsDir)
	if err != nil {
		return fmt.Errorf("Error creating backups `base-dir`. error: %s", err)
	}

	sugar.Infof("Postgres Provider ready to work. Version: %s", info)

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

	azureStorage = flag.Bool("azure-storage", false, "--azure-storage -> dump only the data, not the schema")
	accountName = flag.String("account-name", "", " --account-name -> azure account name")
	accountKey = flag.String("account-key", "", " --account-key -> azure account key")
	containerName = flag.String("container-name", "", " --container-name -> azure container name")

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
	fileString := "--file=" + resolveFilePath(apiID, pgDumpID)

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

	pgDumpCommand := "pg_dump --username=" + *username + " --dbname=" + *dbname + " --host=" + *host + " --port=" + strconv.Itoa(*port) + " --verbose --format=" + backupFormat + " --jobs=1 --compress=9 --column-inserts --inserts --quote-all-identifiers --clean --create " + fileString + " " + dataOnlyString + " " + schemaOnlyString + " " + encodingString
	sugar.Debugf("Executing pg_dump command: %s", pgDumpCommand)
	out, err := schellyhook.ExecShellTimeout(pgDumpCommand, timeout, shellContext)

	if err != nil {
		status := (*shellContext).CmdRef.Status()
		if status.Exit == -1 {
			sugar.Warnf("PostgresProvider pg_dump command timeout enforced (%d seconds)", (status.StopTs-status.StartTs)/1000000000)
		}
		sugar.Debugf("PostgresProvider pg_dump error. out=%s; err=%s", out, err.Error())
		errorFileBytes := []byte(pgDumpID)

		errorFilePath := resolveErrorFilePath(apiID)
		err := ioutil.WriteFile(errorFilePath, errorFileBytes, 0600)
		if err != nil {
			sugar.Errorf("Error writing .error file for %s. err: %s", apiID, err)
			return err
		}

		if *azureStorage {
			err = sendFileToAzure(*accountName, *accountKey, *containerName, resolveErrorFilePathAzure(apiID), resolveErrorFilePath(apiID))
			if err != nil {
				sugar.Debugf("Send error file to Azure with error: %s", err.Error())
				return fmt.Errorf("Send errro file to Azure with error: %s", err.Error())
			}
		}

		return err
	}

	sugar.Debugf("PostgresProvider pg_dump backup started. Output log:")
	sugar.Debugf(out)
	saveDataID(apiID, pgDumpID)

	//## Send file to Azure Storage Blob
	if *azureStorage {
		err = sendFileToAzure(*accountName, *accountKey, *containerName, resolveFilePathAzure(apiID, pgDumpID), resolveFilePathAzure(apiID, pgDumpID))
		if err != nil {
			sugar.Debugf("Send file to Azure with error: %s", err.Error())
			return fmt.Errorf("Send file to Azure with error: %s", err.Error())
		}
	}

	sugar.Infof("Postgres backup launched")
	return nil
}

//GetAllBackups returns all backups from underlaying backuper. optional for Schelly
func (sb PostgresBackuper) GetAllBackups() (result []schellyhook.SchellyResponse, err error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("GetAllBackups")

	if *azureStorage {
		result, err = listFilesFromAzure(*accountName, *accountKey, *containerName)
		if err != nil {
			sugar.Debugf("List files from Azure with error: %s", err.Error())
			return nil, err
		}
	} else {
		files, err := ioutil.ReadDir(*backupsDir)
		if err != nil {
			return nil, err
		}

		backups := make([]schellyhook.SchellyResponse, 0)
		for _, fileName := range files {

			id := strings.Split(fileName.Name(), dataStringSeparator)[1]
			dataID := strings.Split(fileName.Name(), dataStringSeparator)[2]
			sizeMB := fileName.Size()

			backupFilePath := *backupsDir + "/" + fileName.Name()
			_, err = os.Open(backupFilePath)
			if err != nil {
				return nil, err
			}
			sugar.Debugf("Found and opened backup file: %s", backupFilePath)
			status := "available"

			sr := schellyhook.SchellyResponse{
				ID:      id,
				DataID:  dataID,
				Status:  status,
				Message: backupFilePath,
				SizeMB:  float64(sizeMB),
			}
			backups = append(backups, sr)
		}
		result = backups
	}
	return result, nil
}

//GetBackup get an specific backup along with status
func (sb PostgresBackuper) GetBackup(apiID string) (res *schellyhook.SchellyResponse, err error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("GetBackup apiID=%s", apiID)

	if *azureStorage {
		pgDumpID, err0 := getDataIDFromAzure(*accountName, *accountName, *containerName, apiID)
		if err0 != nil {
			sugar.Debugf("Error finding pgDumpID for apiId %s. err=%s", apiID, err0)
			return nil, err0
		}
		res, err = findFileFromAzure(*accountName, *accountKey, *containerName, resolveFilePathAzure(apiID, pgDumpID))
		if err != nil {
			sugar.Debugf("Error finding file with pgDumpID %s for apiId %s. err=%s", pgDumpID, apiID, err)
			return nil, err
		}
	} else {
		pgDumpID, err0 := getDataID(apiID)
		if err0 != nil {
			sugar.Debugf("Error finding pgDumpID for apiId %s. err=%s", apiID, err0)
			return nil, err0
		}
		if pgDumpID == "" {
			sugar.Debugf("pgDumpID not found for apiId %s.", apiID)
			return nil, nil
		}

		sugar.Debugf("Found pgDumpID=" + pgDumpID + " for apiID: " + apiID + ". Finding Backup file...")
		res, err = findBackup(apiID, pgDumpID)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

//DeleteBackup removes current backup from underlaying backup storage
func (sb PostgresBackuper) DeleteBackup(apiID string) error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("DeleteBackup apiID=%s", apiID)

	if *azureStorage {
		errorFilePath := resolveErrorFilePathAzure(apiID)
		_, err := findFileFromAzure(*accountName, *accountKey, *containerName, errorFilePath)
		if err == nil {
			sugar.Debugf("Error file found: %s. The backup %s had problems during execution and will be considered as deleted", errorFilePath, apiID)
			err = deleteFileFromAzure(*accountName, *accountKey, *containerName, errorFilePath)
			if err != nil {
				sugar.Debugf("Deleting backup file with problems %s from azure with error: %s", errorFilePath, err.Error())
				return err
			}
			return nil
		}

		pgDumpID, err0 := getDataIDFromAzure(*accountName, *accountKey, *containerName, apiID)
		if err0 != nil {
			sugar.Debugf("pgDumpID not found for apiId %s. err=%s", apiID, err0)
			return err0
		}

		_, err0 = findFileFromAzure(*accountName, *accountKey, *containerName, resolveFilePathAzure(apiID, pgDumpID))
		if err0 != nil {
			sugar.Debugf("Backup apiID %s, pgDumpID %s not found for removal", apiID, pgDumpID)
			return err0
		}

		err = deleteFileFromAzure(*accountName, *accountKey, *containerName, resolveFilePathAzure(apiID, pgDumpID))
		if err != nil {
			sugar.Debugf("Deleting backup file %s from azure with error: %s", resolveFilePathAzure(apiID, pgDumpID), err.Error())
			return err
		}
	} else {
		errorFilePath := resolveErrorFilePath(apiID)
		_, err := os.Open(errorFilePath)
		if err == nil { //if the file exists, this backup should be discarded
			sugar.Debugf("Error file found: %s. The backup %s had problems during execution and will be considered as deleted", errorFilePath, apiID)
			os.Remove(errorFilePath) //try to remove the file
			return nil
		}

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

		err1 := os.Remove(resolveFilePath(apiID, pgDumpID))
		if err1 != nil {
			return err1
		}
		sugar.Debugf("Delete apiID %s pgDumpID %s successful", apiID, pgDumpID)
	}
	return nil
}

func findBackup(apiID string, pgDumpID string) (*schellyhook.SchellyResponse, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	backupFilePath := resolveFilePath(apiID, pgDumpID)
	result, err := os.Open(backupFilePath)
	if err != nil {
		sugar.Errorf("File " + backupFilePath + " not found")
		return nil, err
	}
	file, err := result.Stat()
	if err != nil {
		return nil, err
	}

	sugar.Debugf("pgDumpID found. Details: %s", file)

	status := "available"

	return &schellyhook.SchellyResponse{
		ID:      apiID,
		DataID:  pgDumpID,
		Status:  status,
		Message: backupFilePath,
		SizeMB:  float64(file.Size()),
	}, nil
}

func getDataID(apiID string) (string, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Debugf("Searching dataID (pgDumpID) for apiID: %s", apiID)
	files, err := ioutil.ReadDir(*backupsDir)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		sugar.Debugf("Backup File <Loop>: %s", file.Name())
		if strings.Contains(file.Name(), apiID) && strings.Contains(file.Name(), dataStringSeparator) {
			if _, err := os.Stat(*backupsDir + "/" + file.Name()); err == nil {
				sugar.Debugf("Found file for apiID reference: %s", apiID)
				_, err0 := ioutil.ReadFile(*backupsDir + "/" + file.Name())
				if err0 != nil {
					return "", err0
				}
				pgDumpID := strings.Split(file.Name(), dataStringSeparator)[2]
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

func resolveFilePath(apiID string, pgDumpID string) string {
	return *backupsDir + "/" + *fileName + dataStringSeparator + apiID + dataStringSeparator + pgDumpID
}

func resolveFilePathAzure(apiID string, pgDumpID string) string {
	return *fileName + dataStringSeparator + apiID + dataStringSeparator + pgDumpID
}

func resolveErrorFilePath(apiID string) string {
	return *backupsDir + "/" + apiID + ".err"
}

func resolveErrorFilePathAzure(apiID string) string {
	return apiID + ".err"
}

func mkDirs(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
}

func handleErrors(err *error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	if *err != nil {
		if serr, ok := (*err).(azblob.StorageError); ok { // This error is a Service-specific
			switch serr.ServiceCode() { // Compare serviceCode to ServiceCodeXxx constants
			case azblob.ServiceCodeContainerAlreadyExists:
				sugar.Debugf("Received 409. Container already exists")
				(*err) = nil
			default:
				sugar.Debugf("Handle Errors: %s", (*err).Error())
			}
		}
	}
}

func connectToAzureContainer(accountName string, accountKey string, containerName string) (azblob.ContainerURL, context.Context, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	// Create a default request pipeline using your storage account name and account key.
	sugar.Debugf("Connecting with Azure -> AccountName: %s", accountName)
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		sugar.Debugf("Invalid credentials with error: %s", err.Error())
		return azblob.ContainerURL{}, nil, fmt.Errorf("Invalid credentials with error: %s", err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// From the Azure portal, get your storage account blob service URL endpoint.
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))

	// Create a ContainerURL object that wraps the container URL and a request
	// pipeline to make requests.
	containerURL := azblob.NewContainerURL(*URL, p)
	ctx := context.Background()

	return containerURL, ctx, nil
}

func sendFileToAzure(accountName string, accountKey string, containerName string, fileName string, filePath string) error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	containerURL, ctx, err := connectToAzureContainer(accountName, accountKey, containerName)
	if err != nil {
		sugar.Debugf("Connect to Azure with error: %s", err.Error())
		return fmt.Errorf("Connect to Azure with error: %s", err.Error())
	}

	_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	handleErrors(&err)
	if err != nil {
		sugar.Debugf("Create Container with error: %s", err.Error())
		return fmt.Errorf("Create Container with error: %s", err.Error())
	}

	// Here's how to upload a blob.
	blobURL := containerURL.NewBlockBlobURL(fileName)
	file, err := os.Open(filePath)
	handleErrors(&err)
	if err != nil {
		sugar.Debugf("Open file with error: %s", err.Error())
		return fmt.Errorf("Open file with error: %s", err.Error())
	}

	// You can use the low-level PutBlob API to upload files. Low-level APIs are simple wrappers for the Azure Storage REST APIs.
	// Note that PutBlob can upload up to 256MB data in one shot. Details: https://docs.microsoft.com/en-us/rest/api/storageservices/put-blob
	// Following is commented out intentionally because we will instead use UploadFileToBlockBlob API to upload the blob
	// _, err = blobURL.PutBlob(ctx, file, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	// handleErrors(err)

	// The high-level API UploadFileToBlockBlob function uploads blocks in parallel for optimal performance, and can handle large files as well.
	// This function calls PutBlock/PutBlockList for files larger 256 MBs, and calls PutBlob for any file smaller
	sugar.Debugf("Uploading the file with blob name: %s\n", fileName)
	_, err = azblob.UploadFileToBlockBlob(ctx, file, blobURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16})
	handleErrors(&err)
	if err != nil {
		sugar.Debugf("Upload file with error: %s", err.Error())
		return fmt.Errorf("Upload file with error: %s", err.Error())
	}

	return nil
}

func deleteFileFromAzure(accountName string, accountKey string, containerName string, fileName string) error {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	containerURL, ctx, err := connectToAzureContainer(accountName, accountKey, containerName)
	if err != nil {
		sugar.Debugf("Connect to Azure with error: %s", err.Error())
		return fmt.Errorf("Connect to Azure with error: %s", err.Error())
	}

	blobURL := containerURL.NewBlockBlobURL(fileName)
	_, err = blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})

	if err != nil {
		sugar.Debugf("Delete file %s at container %s with error: %s", fileName, containerName, err.Error())
		return fmt.Errorf("Delete file %s at container %s with error: %s", fileName, containerName, err.Error())
	}

	return nil
}

func listFilesFromAzure(accountName string, accountKey string, containerName string) ([]schellyhook.SchellyResponse, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	containerURL, ctx, err := connectToAzureContainer(accountName, accountKey, containerName)
	if err != nil {
		sugar.Debugf("Connect to Azure with error: %s", err.Error())
		return nil, fmt.Errorf("Connect to Azure with error: %s", err.Error())
	}

	backups := make([]schellyhook.SchellyResponse, 0)
	for marker := (azblob.Marker{}); marker.NotDone(); {
		// Get a result segment starting with the blob indicated by the current Marker.
		listBlob, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		handleErrors(&err)

		// ListBlobs returns the start of the next segment; you MUST use this to get
		// the next segment (after processing the current result segment).
		marker = listBlob.NextMarker

		// Process the blobs returned in this result segment (if the segment is empty, the loop body won't execute)
		for _, blobInfo := range listBlob.Segment.BlobItems {
			sugar.Debugf("	Blob name: %s", blobInfo.Name)
			id := strings.Split(blobInfo.Name, dataStringSeparator)[1]
			dataID := strings.Split(blobInfo.Name, dataStringSeparator)[2]
			sizeMB := blobInfo.Properties.ContentLength

			blobURL := containerURL.NewBlockBlobURL(blobInfo.Name)
			backupFilePath := blobURL.String()
			// sugar.Debugf("Found and opened backup file: %s", backupFilePath)
			var status string
			if blobInfo.Deleted {
				status = "deleted"
			} else {
				status = "avaliable"
			}

			sr := schellyhook.SchellyResponse{
				ID:      id,
				DataID:  dataID,
				Status:  status,
				Message: backupFilePath,
				SizeMB:  float64(*sizeMB),
			}
			backups = append(backups, sr)
		}

	}

	return backups, nil
}

func getDataIDFromAzure(accountName string, accountKey string, containerName string, apiID string) (string, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	containerURL, ctx, err := connectToAzureContainer(accountName, accountKey, containerName)
	if err != nil {
		sugar.Debugf("Connect to Azure with error: %s", err.Error())
		return "", fmt.Errorf("Connect to Azure with error: %s", err.Error())
	}

	for marker := (azblob.Marker{}); marker.NotDone(); {
		// Get a result segment starting with the blob indicated by the current Marker.
		listBlob, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		handleErrors(&err)

		// ListBlobs returns the start of the next segment; you MUST use this to get
		// the next segment (after processing the current result segment).
		marker = listBlob.NextMarker

		// Process the blobs returned in this result segment (if the segment is empty, the loop body won't execute)
		for _, blobInfo := range listBlob.Segment.BlobItems {
			fmt.Print("	Blob name: " + blobInfo.Name + "\n")
			if strings.Contains(blobInfo.Name, apiID) && strings.Contains(blobInfo.Name, dataStringSeparator) {
				pgDumpID := strings.Split(blobInfo.Name, dataStringSeparator)[2]
				sugar.Debugf("apiID %s <-> pgDumpID %s", apiID, pgDumpID)
				return pgDumpID, nil
			}
		}
	}

	return "", fmt.Errorf("pgDumpID for %s not found", apiID)
}

func findFileFromAzure(accountName string, accountKey string, containerName string, fileName string) (*schellyhook.SchellyResponse, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	containerURL, ctx, err := connectToAzureContainer(accountName, accountKey, containerName)
	if err != nil {
		sugar.Debugf("Connect to Azure with error: %s", err.Error())
		return &schellyhook.SchellyResponse{}, fmt.Errorf("Connect to Azure with error: %s", err.Error())
	}

	blobURL := containerURL.NewBlockBlobURL(fileName)
	blobInfo, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})

	id := strings.Split(fileName, dataStringSeparator)[1]
	dataID := strings.Split(fileName, dataStringSeparator)[2]
	sizeMB := blobInfo.ContentLength()
	backupFilePath := blobURL.String()

	sugar.Debugf("Found and opened backup file: %s", backupFilePath)
	status := blobInfo.Status()

	return &schellyhook.SchellyResponse{
		ID:      id,
		DataID:  dataID,
		Status:  status,
		Message: backupFilePath,
		SizeMB:  float64(sizeMB),
	}, nil
}
