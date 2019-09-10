package main

import (
	"encoding/json"
	"flag"
	"testing"
	"time"

	"go.uber.org/zap"
)

var (
	filenameTest      = "test---12345---20190614092818"
	accountNameTest   = "cephbacky2"
	accountKeyTest    = "Pim4+QnLVPzWNOhxUaftRcIfduK/aEMvpq8ZqMckDs5jv28jadtzX2i80u1V6BMQmgb4ihGrqFVYunrBelnapA=="
	containerNameTest = "ceph-backy2-test"
)

func TestRegisterFlags(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	backuper := PostgresBackuper{}

	sugar.Infof("Starting TestRegisterFlags...")

	err := backuper.RegisterFlags()

	if err != nil {
		t.Errorf("Error registering flags: %s", err)
	}

	requiredFlags := []string{"file", "dbname", "host", "port", "username", "password"}
	for _, f := range requiredFlags {
		if flag.Lookup(f) == nil {
			t.Errorf("Imperative flag not defined: %s", f)
		} else {
			sugar.Infof("File flag: %s", flag.Lookup(f))
		}
	}
}

func TestCreateFileName(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	fileName = &filenameTest
	dataStringSeparator = "---"
	sugar.Infof("Starting TestSendFileToAzure...")
	pgDumpID := time.Now().Format("20060102150405")
	result := resolveFilePathAzure("12345", pgDumpID)
	sugar.Debugf("Filename: %s", result)
}

func TestSendFileToAzure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infof("Starting TestSendFileToAzure...")

	err := sendFileToAzure(accountNameTest, accountKeyTest, containerNameTest, filenameTest, "./"+filenameTest)
	if err != nil {
		sugar.Infof("Test send file to azure with error!")
		sugar.Infof("%s", err.Error())
		panic(err)
	}

	sugar.Infof("(V) Test send file to azure !")
}

func TestListFilesFromAzure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infof("Starting TestListFilesFromAzure...")
	file := "test"
	fileName = &file
	dataStringSeparator = "---"

	resp, err := listFilesFromAzure(accountNameTest, accountKeyTest, containerNameTest)
	if err != nil {
		sugar.Infof("Test list files from azure with error!")
		sugar.Infof("%s", err.Error())
		panic(err)
	}

	sugar.Debugf("Listing Files at Container %s ", containerNameTest)
	for idx, fileItem := range resp {
		jsonItem, _ := json.Marshal(fileItem)
		sugar.Debugf("%d of %d -> %s", idx+1, len(resp), string(jsonItem))
	}

	sugar.Infof("(V) Test list files from azure !")
}

func TestGetFileInfoFromAzure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infof("Starting TestGetFileInfoFromAzure...")
	file := "test"
	fileName = &file
	dataStringSeparator = "---"

	resp, err := getDataIDFromAzure(accountNameTest, accountKeyTest, containerNameTest, "12345")
	if err != nil {
		sugar.Infof("Test list files from azure with error!")
		sugar.Infof("%s", err.Error())
		panic(err)
	}
	respInfo, err := findFileFromAzure(accountNameTest, accountKeyTest, containerNameTest, resolveFilePathAzure("12345", resp))
	if err != nil {
		sugar.Infof("Test list files from azure with error!")
		sugar.Infof("%s", err.Error())
		panic(err)
	}

	sugar.Debugf("File %s info at Container %s ", filenameTest, containerNameTest)
	jsonItem, _ := json.Marshal(respInfo)
	sugar.Debugf("%s", string(jsonItem))

	sugar.Infof("(V) Test get file info from azure !")
}

func TestDeleteFileFromAzure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Infof("Starting TestDeleteFileFromAzure...")

	err := deleteFileFromAzure(accountNameTest, accountKeyTest, containerNameTest, filenameTest)
	if err != nil {
		sugar.Infof("Test delete file from azure with error!")
		sugar.Infof("%s", err.Error())
		panic(err)
	}

	sugar.Infof("(V) Test delete file from azure !")
}
