package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
)

func processComplexField(s string) string {
	return html.UnescapeString(s)
}

//-- Function to Load Configuration File
func loadConfig() AzureImportConfStruct {
	//-- Check Config File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}

	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
	}
	//-- New Decoder
	decoder := json.NewDecoder(file)

	eldapConf := AzureImportConfStruct{}

	//-- Decode JSON
	err := decoder.Decode(&eldapConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
	}

	//-- Return New Congfig
	return eldapConf
}

func validateConf() error {
	//-- Check for API Key
	if AzureImportConf.APIKey == "" {
		err := errors.New("API Key is not set")
		return err
	}
	//-- Check for Instance ID
	if AzureImportConf.InstanceID == "" {
		err := errors.New("InstanceID is not set")
		return err
	}
	return nil
}
