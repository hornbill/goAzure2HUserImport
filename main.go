package main

//----- Packages -----
import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"time"

	"github.com/hornbill/color"
	apiLib "github.com/hornbill/goApiLib"
)

//----- Main Function -----
func main() {
	//-- Initiate Variables
	initVars()

	//-- Process Flags
	procFlags()

	//-- If configVersion just output version number and exit
	if configVersion {
		fmt.Printf("%v \n", version)
		return
	}

	//-- Load Configuration File Into Struct
	AzureImportConf = loadConfig()

	//-- Validation on Configuration File
	err := validateConf()
	if err != nil {
		logger(4, fmt.Sprintf("%v", err), true)
		logger(4, "Please Check your Configuration File: "+configFileName, true)
		return
	}

	//Sort out password profile
	getPasswordProfile()

	//New XMLMC instance to get DAV endpoint and pass to espLogger
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.InstanceID)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)

	//-- Get Instance XMLMC Endpoint
	AzureImportConf.DAVURL = espXmlmc.DavEndpoint
	logger(1, "Instance DAV Endpoint "+fmt.Sprintf("%v", AzureImportConf.DAVURL), true)

	//-- Once we have loaded the config write to hornbill log file
	logged := espLogger("---- XMLMC Azure Import Utility V"+fmt.Sprintf("%v", version)+" ----", "debug", espXmlmc)

	if !logged {
		logger(4, "Unable to Connect to Instance", true)
		return
	}

	if AzureImportConf.AzureConf.Search == "users" {
		//Get users, process accordingly
		logger(2, "Querying Users with filter ["+AzureImportConf.AzureConf.UserFilter+"]", true)
		for ok := true; ok; ok = (strAzurePagerToken != "") {
			var boolSQLUsers, arrUsers = queryUsers()
			if boolSQLUsers {
				processUsers(arrUsers)
			} else {
				logger(3, "No (further) Users found in User search", true)
				return
			}
		}
	}
	if AzureImportConf.AzureConf.Search == "groups" {
		for _, group := range AzureImportConf.AzureConf.UsersByGroupID {
			//Get groups, process accordingly
			logger(2, "Querying Group ["+group.Name+"]", true)

			for ok := true; ok; ok = (strAzurePagerToken != "") {
				var boolSQLUsers, arrUsers = queryGroup(group.ObjectID)
				if boolSQLUsers {
					processUsers(arrUsers)
				} else {
					logger(3, "No (further) Users found in Group ["+group.Name+"]", true)
				}
			}
		}
	}
	outputEnd(espXmlmc)
}

func initVars() {
	//-- Start Time for Durration
	startTime = time.Now()
	//-- Start Time for Log File
	timeNow = time.Now().Format(time.RFC3339)
	//-- Remove :
	timeNow = strings.Replace(timeNow, ":", "-", -1)
	//-- Set Counter
	errorCount = 0
}

func procFlags() {
	//-- Grab Flags
	flag.StringVar(&configFileName, "file", "conf.json", "Name of Configuration File To Load")
	flag.StringVar(&configLogPrefix, "logprefix", "", "Add prefix to the logfile")
	flag.BoolVar(&configDebug, "debug", false, "Enable additional debug logging")
	flag.BoolVar(&configDryRun, "dryrun", false, "Allow the Import to run without Creating or Updating users")
	flag.BoolVar(&configVersion, "version", false, "Output Version")
	flag.IntVar(&configMaxRoutines, "concurrent", 1, "Maximum number of users to import concurrently.")

	//-- Parse Flags
	flag.Parse()

	//-- Output config
	if !configVersion {
		outputFlags()
	}

	if configMaxRoutines < 1 || configMaxRoutines > 10 {
		color.Red("The maximum concurrent requests allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + strconv.Itoa(configMaxRoutines) + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}
}

func outputEnd(espXmlmc *apiLib.XmlmcInstStruct) {
	//-- End output
	if errorCount > 0 {
		logger(4, "Error encountered please check the log file", true)
		logger(4, "Error Count: "+fmt.Sprintf("%d", errorCount), true)
	}
	logger(1, "Updated: "+fmt.Sprintf("%d", counters.updated), true)
	logger(1, "Updated Skipped: "+fmt.Sprintf("%d", counters.updatedSkipped), true)

	logger(1, "Created: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Created Skipped: "+fmt.Sprintf("%d", counters.createskipped), true)

	logger(1, "Profiles Updated: "+fmt.Sprintf("%d", counters.profileUpdated), true)
	logger(1, "Profiles Skipped: "+fmt.Sprintf("%d", counters.profileSkipped), true)

	//-- Show Time Takens
	endTime = time.Since(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	//-- complete
	complete(espXmlmc)
	logger(1, "---- XMLMC Azure Import Complete ---- ", true)
}

func outputFlags() {
	//-- Output
	logger(1, "---- XMLMC Azure Import Utility V"+fmt.Sprintf("%v", version)+" ----", true)

	logger(1, "Flag - Config File "+configFileName, true)
	logger(1, "Flag - Log Prefix "+configLogPrefix, true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)
	logger(1, "Flag - Workers "+fmt.Sprintf("%v", configMaxRoutines), false)
}

//-- complete
func complete(espXmlmc *apiLib.XmlmcInstStruct) {
	//-- End output
	espLogger("Errors: "+fmt.Sprintf("%d", errorCount), "error", espXmlmc)
	espLogger("Updated: "+fmt.Sprintf("%d", counters.updated), "debug", espXmlmc)
	espLogger("Updated Skipped: "+fmt.Sprintf("%d", counters.updatedSkipped), "debug", espXmlmc)
	espLogger("Created: "+fmt.Sprintf("%d", counters.created), "debug", espXmlmc)
	espLogger("Created Skipped: "+fmt.Sprintf("%d", counters.createskipped), "debug", espXmlmc)
	espLogger("Profiles Updated: "+fmt.Sprintf("%d", counters.profileUpdated), "debug", espXmlmc)
	espLogger("Profiles Skipped: "+fmt.Sprintf("%d", counters.profileSkipped), "debug", espXmlmc)
	espLogger("Time Taken: "+fmt.Sprintf("%v", endTime), "debug", espXmlmc)
	espLogger("---- XMLMC Azure User Import Complete ---- ", "debug", espXmlmc)
}
