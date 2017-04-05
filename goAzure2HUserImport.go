package main

//----- Packages -----
import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"html"
	"log"
	"os"
	"text/template"
	/* DAV inclusion */
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"crypto/rand"
	"github.com/hornbill/color" //-- CLI Colour
	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb" //--Hornbil Clone of "github.com/cheggaaa/pb"
	"strconv"
	"strings"
	"sync"
	"time"
)

//----- Constants -----
const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	version      = "1.0.0"
	constOK      = "ok"
	updateString = "Update"
	createString = "Create"
)

var (
	AzureImportConf     AzureImportConfStruct
	xmlmcInstanceConfig xmlmcConfig
	xmlmcUsers          []userListItemStruct
	sites               []siteListStruct
	managers            []managerListStruct
	groups              []groupListStruct
	counters            counterTypeStruct
	configFileName      string
	configZone          string
	configLogPrefix     string
	configDryRun        bool
	configVersion       bool
	configWorkers       int
	configMaxRoutines   string
	timeNow             string
	startTime           time.Time
	endTime             time.Duration
	errorCount          uint64
	noValuesToUpdate    = "There are no values to update"
	mutex               = &sync.Mutex{}
	mutexBar            = &sync.Mutex{}
	mutexCounters       = &sync.Mutex{}
	mutexCustomers      = &sync.Mutex{}
	mutexSite           = &sync.Mutex{}
	mutexSites          = &sync.Mutex{}
	mutexGroups         = &sync.Mutex{}
	mutexManagers       = &sync.Mutex{}
	logFileMutex        = &sync.Mutex{}
	bufferMutex         = &sync.Mutex{}
	worker              sync.WaitGroup
	maxGoroutines       = 6

	userProfileMappingMap = map[string]string{
		"MiddleName":        "middleName",
		"JobDescription":    "jobDescription",
		"Manager":           "manager",
		"WorkPhone":         "workPhone",
		"Qualifications":    "qualifications",
		"Interests":         "interests",
		"Expertise":         "expertise",
		"Gender":            "gender",
		"Dob":               "dob",
		"Nationality":       "nationality",
		"Religion":          "religion",
		"HomeTelephone":     "homeTelephone",
		"SocialNetworkA":    "socialNetworkA",
		"SocialNetworkB":    "socialNetworkB",
		"SocialNetworkC":    "socialNetworkC",
		"SocialNetworkD":    "socialNetworkD",
		"SocialNetworkE":    "socialNetworkE",
		"SocialNetworkF":    "socialNetworkF",
		"SocialNetworkG":    "socialNetworkG",
		"SocialNetworkH":    "socialNetworkH",
		"PersonalInterests": "personalInterests",
		"HomeAddress":       "homeAddress",
		"PersonalBlog":      "personalBlog",
		"Attrib1":           "attrib1",
		"Attrib2":           "attrib2",
		"Attrib3":           "attrib3",
		"Attrib4":           "attrib4",
		"Attrib5":           "attrib5",
		"Attrib6":           "attrib6",
		"Attrib7":           "attrib7",
		"Attrib8":           "attrib8"}

	userProfileArray = []string{
		"MiddleName",
		"JobDescription",
		"Manager",
		"WorkPhone",
		"Qualifications",
		"Interests",
		"Expertise",
		"Gender",
		"Dob",
		"Nationality",
		"Religion",
		"HomeTelephone",
		"SocialNetworkA",
		"SocialNetworkB",
		"SocialNetworkC",
		"SocialNetworkD",
		"SocialNetworkE",
		"SocialNetworkF",
		"SocialNetworkG",
		"SocialNetworkH",
		"PersonalInterests",
		"HomeAddress",
		"PersonalBlog",
		"Attrib1",
		"Attrib2",
		"Attrib3",
		"Attrib4",
		"Attrib5",
		"Attrib6",
		"Attrib7",
		"Attrib8"}

	userMappingMap = map[string]string{
		"Name":           "name",
		"Password":       "password",
		"UserType":       "userType",
		"FirstName":      "firstName",
		"LastName":       "lastName",
		"JobTitle":       "jobTitle",
		"Site":           "site",
		"Phone":          "phone",
		"Email":          "email",
		"Mobile":         "mobile",
		"AbsenceMessage": "absenceMessage",
		"TimeZone":       "timeZone",
		"Language":       "language",
		"DateTimeFormat": "dateTimeFormat",
		"DateFormat":     "dateFormat",
		"TimeFormat":     "timeFormat",
		"CurrencySymbol": "currencySymbol",
		"CountryCode":    "countryCode"}

	userUpdateArray = []string{
		"userId",
		"UserType",
		"Name",
		"Password",
		"FirstName",
		"LastName",
		"JobTitle",
		"Site",
		"Phone",
		"Email",
		"Mobile",
		"AbsenceMessage",
		"TimeZone",
		"Language",
		"DateTimeFormat",
		"DateFormat",
		"TimeFormat",
		"CurrencySymbol",
		"CountryCode"}

	userCreateArray = []string{
		"userId",
		"Name",
		"Password",
		"UserType",
		"FirstName",
		"LastName",
		"JobTitle",
		"Site",
		"Phone",
		"Email",
		"Mobile",
		"AbsenceMessage",
		"TimeZone",
		"Language",
		"DateTimeFormat",
		"DateFormat",
		"TimeFormat",
		"CurrencySymbol",
		"CountryCode"}
)

type siteListStruct struct {
	SiteName string
	SiteID   int
}
type xmlmcSiteListResponse struct {
	MethodResult string               `xml:"status,attr"`
	Params       paramsSiteListStruct `xml:"params"`
	State        stateStruct          `xml:"state"`
}
type paramsSiteListStruct struct {
	RowData paramsSiteRowDataListStruct `xml:"rowData"`
}
type paramsSiteRowDataListStruct struct {
	Row siteObjectStruct `xml:"row"`
}
type siteObjectStruct struct {
	SiteID      int    `xml:"h_id"`
	SiteName    string `xml:"h_site_name"`
	SiteCountry string `xml:"h_country"`
}

type managerListStruct struct {
	UserName string
	UserID   string
}
type groupListStruct struct {
	GroupName string
	GroupID   string
}

type xmlmcConfig struct {
	instance string
	zone     string
	url      string
}

type counterTypeStruct struct {
	updated        uint16
	created        uint16
	profileUpdated uint16
	updatedSkipped uint16
	createskipped  uint16
	profileSkipped uint16
}
type userMappingStruct struct {
	UserID         string
	UserType       string
	Name           string
	Password       string
	FirstName      string
	LastName       string
	JobTitle       string
	Site           string
	Phone          string
	Email          string
	Mobile         string
	AbsenceMessage string
	TimeZone       string
	Language       string
	DateTimeFormat string
	DateFormat     string
	TimeFormat     string
	CurrencySymbol string
	CountryCode    string
}
type userAccountStatusStruct struct {
	Action  string
	Enabled bool
	Status  string
}
type userProfileMappingStruct struct {
	MiddleName        string
	JobDescription    string
	Manager           string
	WorkPhone         string
	Qualifications    string
	Interests         string
	Expertise         string
	Gender            string
	Dob               string
	Nationality       string
	Religion          string
	HomeTelephone     string
	SocialNetworkA    string
	SocialNetworkB    string
	SocialNetworkC    string
	SocialNetworkD    string
	SocialNetworkE    string
	SocialNetworkF    string
	SocialNetworkG    string
	SocialNetworkH    string
	PersonalInterests string
	HomeAddress       string
	PersonalBlog      string
	Attrib1           string
	Attrib2           string
	Attrib3           string
	Attrib4           string
	Attrib5           string
	Attrib6           string
	Attrib7           string
	Attrib8           string
}
type userManagerStruct struct {
	Action  string
	Enabled bool
}

type siteLookupStruct struct {
	Action    string
	Enabled   bool
	Attribute string
}
type imageLinkStruct struct {
	Action     string
	Enabled    bool
	UploadType string
	ImageType  string
	URI        string
}
type orgLookupStruct struct {
	Action   string
	Enabled  bool
	OrgUnits []OrgUnitStruct
}
type OrgUnitStruct struct {
	Attribute   string
	Type        int
	Membership  string
	TasksView   bool
	TasksAction bool
}
type AzureImportConfStruct struct {
	APIKey             string
	InstanceID         string
	URL                string
	DAVURL             string
	UpdateUserType     bool
	UserRoleAction     string
	UserIdentifier     string
	AzureConf          azureConfStruct
	UserMapping        map[string]string //userMappingStruct
	UserAccountStatus  userAccountStatusStruct
	UserProfileMapping map[string]string //userProfileMappingStruct
	UserManagerMapping userManagerStruct
	Roles              []string
	SiteLookup         siteLookupStruct
	ImageLink          imageLinkStruct
	OrgLookup          orgLookupStruct
}
type xmlmcResponse struct {
	MethodResult string       `xml:"status,attr"`
	Params       paramsStruct `xml:"params"`
	State        stateStruct  `xml:"state"`
}
type xmlmcCheckUserResponse struct {
	MethodResult string                 `xml:"status,attr"`
	Params       paramsCheckUsersStruct `xml:"params"`
	State        stateStruct            `xml:"state"`
}
type xmlmcUserListResponse struct {
	MethodResult string                     `xml:"status,attr"`
	Params       paramsUserSearchListStruct `xml:"params"`
	State        stateStruct                `xml:"state"`
}
type paramsUserSearchListStruct struct {
	RowData paramsUserRowDataListStruct `xml:"rowData"`
}
type paramsUserRowDataListStruct struct {
	Row userObjectStruct `xml:"row"`
}
type userObjectStruct struct {
	UserID   string `xml:"h_user_id"`
	UserName string `xml:"h_name"`
}

type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}
type paramsCheckUsersStruct struct {
	RecordExist bool `xml:"recordExist"`
}
type paramsStruct struct {
	SessionID string `xml:"sessionId"`
}
type paramsUserListStruct struct {
	UserListItem []userListItemStruct `xml:"userListItem"`
}
type userListItemStruct struct {
	UserID string `xml:"userId"`
	Name   string `xml:"name"`
}

//###
type azureConfStruct struct {
	Tenant       string
	ClientID     string
	ClientSecret string
	Filter       string
	UserID       string
	Debug        bool
}

//### organisation units structures
type xmlmcuserSetGroupOptionsResponse struct {
	MethodResult string      `xml:"status,attr"`
	State        stateStruct `xml:"state"`
}
type xmlmcprofileSetImageResponse struct {
	MethodResult string                `xml:"status,attr"`
	Params       paramsGroupListStruct `xml:"params"`
	State        stateStruct           `xml:"state"`
}
type xmlmcGroupListResponse struct {
	MethodResult string                `xml:"status,attr"`
	Params       paramsGroupListStruct `xml:"params"`
	State        stateStruct           `xml:"state"`
}

type paramsGroupListStruct struct {
	RowData paramsGroupRowDataListStruct `xml:"rowData"`
}

type paramsGroupRowDataListStruct struct {
	Row groupObjectStruct `xml:"row"`
}

type groupObjectStruct struct {
	GroupID   string `xml:"h_id"`
	GroupName string `xml:"h_name"`
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

//----- Main Function -----
func main() {

	//-- Initiate Variables
	initVars()

	//-- Process Flags
	procFlags()

	//-- If configVersion just output version number and die
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
		logger(4, "Please Check your Configuration File: "+fmt.Sprintf("%s", configFileName), true)
		return
	}

	//-- Set Instance ID
	var boolSetInstance = setInstance(configZone, AzureImportConf.InstanceID)
	if boolSetInstance != true {
		return
	}

	//-- Generate Instance XMLMC Endpoint
	AzureImportConf.URL = getInstanceURL()
	AzureImportConf.DAVURL = getInstanceDAVURL()
	logger(1, "Instance Endpoint "+fmt.Sprintf("%v", AzureImportConf.URL), true)
	//-- Once we have loaded the config write to hornbill log file
	logged := espLogger("---- XMLMC Azure Import Utility V"+fmt.Sprintf("%v", version)+" ----", "debug")

	if !logged {
		logger(4, "Unable to Connect to Instance", true)
		return
	}

	//Get asset types, process accordingly
	var boolSQLUsers, arrUsers = queryDatabase()
	if boolSQLUsers {
		processUsers(arrUsers)
	} else {
		logger(4, "No Results found", true)
		return
	}

	outputEnd()
}

func outputEnd() {
	//-- End output
	if errorCount > 0 {
		logger(4, "Error encountered please check the log file", true)
		logger(4, "Error Count: "+fmt.Sprintf("%d", errorCount), true)
		//logger(4, "Check Log File for Details", true)
	}
	logger(1, "Updated: "+fmt.Sprintf("%d", counters.updated), true)
	logger(1, "Updated Skipped: "+fmt.Sprintf("%d", counters.updatedSkipped), true)

	logger(1, "Created: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Created Skipped: "+fmt.Sprintf("%d", counters.createskipped), true)

	logger(1, "Profiles Updated: "+fmt.Sprintf("%d", counters.profileUpdated), true)
	logger(1, "Profiles Skipped: "+fmt.Sprintf("%d", counters.profileSkipped), true)

	//-- Show Time Takens
	endTime = time.Now().Sub(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	//-- complete
	complete()
	logger(1, "---- XMLMC Azure Import Complete ---- ", true)
}
func procFlags() {
	//-- Grab Flags
	flag.StringVar(&configFileName, "file", "conf.json", "Name of Configuration File To Load")
	flag.StringVar(&configZone, "zone", "eur", "Override the default Zone the instance sits in")
	flag.StringVar(&configLogPrefix, "logprefix", "", "Add prefix to the logfile")
	flag.BoolVar(&configDryRun, "dryrun", false, "Allow the Import to run without Creating or Updating users")
	flag.BoolVar(&configVersion, "version", false, "Output Version")
	flag.IntVar(&configWorkers, "workers", 1, "Number of Worker threads to use")
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of requests to import concurrently.")

	//-- Parse Flags
	flag.Parse()

	//-- Output config
	if !configVersion {
		outputFlags()
	}

	//Check maxGoroutines for valid value
	maxRoutines, err := strconv.Atoi(configMaxRoutines)
	if err != nil {
		color.Red("Unable to convert maximum concurrency of [" + configMaxRoutines + "] to type INT for processing")
		return
	}
	maxGoroutines = maxRoutines

	if maxGoroutines < 1 || maxGoroutines > 10 {
		color.Red("The maximum concurrent requests allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + configMaxRoutines + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}
}
func outputFlags() {
	//-- Output
	logger(1, "---- XMLMC Azure Import Utility V"+fmt.Sprintf("%v", version)+" ----", true)

	logger(1, "Flag - Config File "+fmt.Sprintf("%s", configFileName), true)
	logger(1, "Flag - Zone "+fmt.Sprintf("%s", configZone), true)
	logger(1, "Flag - Log Prefix "+fmt.Sprintf("%s", configLogPrefix), true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)
	logger(1, "Flag - Workers "+fmt.Sprintf("%v", configWorkers), false)
}

//-- Check Latest
//-- Function to Load Configruation File
func loadConfig() AzureImportConfStruct {
	//-- Check Config File File Exists
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

	//-- Process Config File

	return nil
}

//-- Worker Pool Function
func loggerGen(t int, s string) string {

	var errorLogPrefix = ""
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = "[WARN] "
	case 4:
		errorLogPrefix = "[ERROR] "
	}
	currentTime := time.Now().UTC()
	time := currentTime.Format("2006/01/02 15:04:05")
	return time + " " + errorLogPrefix + s + "\n"
}
func loggerWriteBuffer(s string) {
	logger(0, s, false)
}

//-- Logging function
func logger(t int, s string, outputtoCLI bool) {
	//-- Curreny WD
	cwd, _ := os.Getwd()
	//-- Log Folder
	logPath := cwd + "/log"
	//-- Log File
	logFileName := logPath + "/" + configLogPrefix + "Azure_User_Import_" + timeNow + ".log"
	red := color.New(color.FgRed).PrintfFunc()
	orange := color.New(color.FgCyan).PrintfFunc()
	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			fmt.Printf("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Printf("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	// don't forget to close it
	defer f.Close()
	// assign it to the standard logger
	logFileMutex.Lock()
	log.SetOutput(f)
	logFileMutex.Unlock()
	var errorLogPrefix = ""
	//-- Create Log Entry
	switch t {
	case 0:
		errorLogPrefix = ""
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = "[WARN] "
	case 4:
		errorLogPrefix = "[ERROR] "
	}
	if outputtoCLI {
		if t == 3 {
			orange(errorLogPrefix + s + "\n")
		} else if t == 4 {
			red(errorLogPrefix + s + "\n")
		} else {
			fmt.Printf(errorLogPrefix + s + "\n")
		}

	}
	log.Println(errorLogPrefix + s)
}

//-- complete
func complete() {
	//-- End output
	espLogger("Errors: "+fmt.Sprintf("%d", errorCount), "error")
	espLogger("Updated: "+fmt.Sprintf("%d", counters.updated), "debug")
	espLogger("Updated Skipped: "+fmt.Sprintf("%d", counters.updatedSkipped), "debug")
	espLogger("Created: "+fmt.Sprintf("%d", counters.created), "debug")
	espLogger("Created Skipped: "+fmt.Sprintf("%d", counters.createskipped), "debug")
	espLogger("Profiles Updated: "+fmt.Sprintf("%d", counters.profileUpdated), "debug")
	espLogger("Profiles Skipped: "+fmt.Sprintf("%d", counters.profileSkipped), "debug")
	espLogger("Time Taken: "+fmt.Sprintf("%v", endTime), "debug")
	espLogger("---- XMLMC Azure User Import Complete ---- ", "debug")
}

// Set Instance Id
func setInstance(strZone string, instanceID string) bool {
	//-- Set Zone
	setZone(strZone)
	//-- Check for blank instance
	if instanceID == "" {
		logger(4, "InstanceId Must be Specified in the Configuration File", true)
		return false
	}
	//-- Set Instance
	xmlmcInstanceConfig.instance = instanceID
	return true
}

// Set Instance Zone to Overide Live
func setZone(zone string) {
	xmlmcInstanceConfig.zone = zone

	return
}

//-- Log to ESP
func espLogger(message string, severity string) bool {
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)
	espXmlmc.SetParam("fileName", "Azure_User_Import")
	espXmlmc.SetParam("group", "general")
	espXmlmc.SetParam("severity", severity)
	espXmlmc.SetParam("message", message)

	XMLLogger, xmlmcErr := espXmlmc.Invoke("system", "logMessage")
	var xmlRespon xmlmcResponse
	if xmlmcErr != nil {
		logger(4, "Unable to write to log "+fmt.Sprintf("%s", xmlmcErr), true)
		return false
	}
	err := xml.Unmarshal([]byte(XMLLogger), &xmlRespon)
	if err != nil {
		logger(4, "Unable to write to log "+fmt.Sprintf("%s", err), true)
		return false
	}
	if xmlRespon.MethodResult != constOK {
		logger(4, "Unable to write to log "+xmlRespon.State.ErrorRet, true)
		return false
	}

	return true
}

//-- Function Builds XMLMC End Point
func getInstanceURL() string {
	xmlmcInstanceConfig.url = "https://"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.zone
	xmlmcInstanceConfig.url += "api.hornbill.com/"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.instance
	xmlmcInstanceConfig.url += "/xmlmc/"

	return xmlmcInstanceConfig.url
}

//-- Function Builds XMLMC End Point
func getInstanceDAVURL() string {
	xmlmcInstanceConfig.url = "https://"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.zone
	xmlmcInstanceConfig.url += "api.hornbill.com/"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.instance
	xmlmcInstanceConfig.url += "/dav/"

	return xmlmcInstanceConfig.url
}

func getBearerToken() (string, error) {

	strClientID := AzureImportConf.AzureConf.ClientID
	strClientSecret := AzureImportConf.AzureConf.ClientSecret
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_secret", strClientSecret)
	data.Set("client_id", strClientID)
	//    data.Set("resource", "https://graph.microsoft.com")
	data.Set("resource", "https://graph.windows.net")
	//fmt.Println(data.Encode())
	strData := data.Encode()
	//strClientSecret := url.Query
	strTentant := AzureImportConf.AzureConf.Tenant
	strURL := "https://login.microsoftonline.com/" + strTentant + "/oauth2/token"
	//strData := "grant_type=client_credentials&client_id=" + strClientID +"&client_secret=" + strClientSecret + "&resource=https%3A%2F%2Fgraph.microsoft.com"

	var xmlmcstr = []byte(strData)
	req, err := http.NewRequest("POST", strURL, bytes.NewBuffer(xmlmcstr))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	req.Header.Set("Accept", "text/json")
	duration := time.Second * time.Duration(30)

	//var httpConn *http.Transport
	client := &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}, Timeout: duration}
	resp, err := client.Do(req)
	if err != nil {
		//fmt.Println("YIKES 1")
		return "", err
	}
	//statuscode := resp.StatusCode
	//fmt.Println(statuscode)
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		//fmt.Println("YIKES 2")
		return "", err
	}
	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Println("YIKES 3")
		return "", errors.New("Cant read the body of the response")
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)

	if qerr != nil {
		return "", errors.New("Cant read the JSON")
	}

	q := f.(map[string]interface{})

	strBearerToken := q["access_token"].(string)

	return strBearerToken, nil
}

//queryDatabase -- Query Asset Database for assets of current type
//-- Builds map of assets, returns true if successful
func queryDatabase() (bool, []map[string]interface{}) {
	//Clear existing Asset Map down
	ArrUserMaps := make([]map[string]interface{}, 0)

	logger(3, "[Azure] Query Azure Data using Graph API. Please wait...", true)

	strBearerToken, err := getBearerToken()
	//    fmt.Println(strBearerToken)
	if err != nil || strBearerToken == "" {
		logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrUserMaps
	}

	//strURL := "https://graph.microsoft.com/v1.0/users"
	strTentant := AzureImportConf.AzureConf.Tenant
	strURL := "https://graph.windows.net/" + strTentant + "/users?" //api-version=1.6&%24filter=startswith(displayName%2C%27Mary%27)"

	data := url.Values{}
	data.Set("api-version", "1.6")
	strFilter := AzureImportConf.AzureConf.Filter
	if strFilter != "" {
		data.Set("$filter", strFilter)
	}
	strData := data.Encode()
	strURL += strData
	//fmt.Println(strURL)
	req, err := http.NewRequest("GET", strURL, nil) //, bytes.NewBuffer(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	req.Header.Set("Authorization", "Bearer "+strBearerToken)
	req.Header.Set("Accept", "application/json")
	duration := time.Second * time.Duration(30)

	client := &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}, Timeout: duration}
	resp, err := client.Do(req)
	if err != nil {
		logger(4, " [Azure] Connection Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrUserMaps
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		logger(4, " [Azure] Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrUserMaps
	}
	logger(3, "[Azure] Connection Successful", true)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger(4, " [Azure] Cannot read the body of the response", true)
		return false, ArrUserMaps
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)
	if qerr != nil {
		logger(4, " [Azure] Cannot read the JSON", true)
		return false, ArrUserMaps
	}

	//Build map full of assets
	intUserCount := 0

	q := f.(map[string]interface{})
	if aResults, ok := q["value"].([]interface{}); ok {

		for _, userDetails := range aResults {
			intUserCount++

			blubber := userDetails.(map[string]interface{})
			ArrUserMaps = append(ArrUserMaps, blubber)
			/*
			   for k, v := range blubber {
			       switch vv := v.(type) {
			       case string:
			           fmt.Println(k, "is string", vv)
			       case int:
			           fmt.Println(k, "is int", vv)
			       case []interface{}:
			           fmt.Println(k, "is an array:")
			           for i, u := range vv {
			               fmt.Println(i, u)
			           }
			       default:
			           fmt.Println(k, "is of a type I don't know how to handle")
			       }
			   }
			*/
			/*
			   strLDAPyField := "displayName"
			   if val, ok := blubber[strLDAPyField]; ok {
			       if "" != val && val != nil {
			           fmt.Println(val)
			       } else {
			           fmt.Println("Skipping Empty : " + strLDAPyField)
			       }
			   } else {
			       fmt.Println(strLDAPyField)
			   }

			   strLDAPyField = "surname"
			   if val, ok := blubber[strLDAPyField]; ok {
			       if "" != val && val != nil {
			           fmt.Println(val)
			       } else {
			           fmt.Println("Skipping Empty : " + strLDAPyField)
			       }
			   } else {
			       fmt.Println(strLDAPyField)
			   }
			*/
		}
	}

	/*
		for rows.Next() {
			intUserCount++
			results := make(map[string]interface{})
			err = rows.MapScan(results)
			//Stick marshalled data map in to parent slice
			ArrUserMaps = append(ArrUserMaps, results)
		}
	*/
	logger(3, fmt.Sprintf("[Azure] Found %d results", intUserCount), false)
	return true, ArrUserMaps
}

//processAssets -- Processes Assets from Asset Map
//--If asset already exists on the instance, update
//--If asset doesn't exist, create
func processUsers(arrUsers []map[string]interface{}) {
	bar := pb.StartNew(len(arrUsers))
	logger(1, "Processing Users", false)

	//Get the identity of the AssetID field from the config
	userIDField := fmt.Sprintf("%v", AzureImportConf.AzureConf.UserID)
	//-- Loop each asset
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)

	for _, customerRecord := range arrUsers {
		maxGoroutinesGuard <- struct{}{}
		worker.Add(1)
		userMap := customerRecord
		//Get the asset ID for the current record
		userID := fmt.Sprintf("%s", userMap[userIDField])
		logger(1, "User ID: "+userID, false)
		if userID != "" {
			//logger(1, "User ID: "+fmt.Sprintf("%s", userID), false)
			espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
			espXmlmc.SetAPIKey(AzureImportConf.APIKey)
			go func() {
				defer worker.Done()
				time.Sleep(1 * time.Millisecond)
				mutexBar.Lock()
				bar.Increment()
				mutexBar.Unlock()

				var boolUpdate = false
				var isErr = false
				boolUpdate, err := checkUserOnInstance(userID, espXmlmc)
				if err != nil {
					logger(4, "Unable to Search For User: "+fmt.Sprintf("%v", err), true)
					isErr = true
				}
				//-- Update or Create Asset
				if !isErr {
					if boolUpdate {
						logger(1, "Update Customer: "+userID, false)
						_, errUpdate := updateUser(userMap, espXmlmc)
						if errUpdate != nil {
							logger(4, "Unable to Update User: "+fmt.Sprintf("%v", errUpdate), false)
						}
					} else {
						logger(1, "Create Customer: "+userID, false)
						_, errorCreate := createUser(userMap, espXmlmc)
						if errorCreate != nil {
							logger(4, "Unable to Create User: "+fmt.Sprintf("%v", errorCreate), false)
						}
					}
				}
				<-maxGoroutinesGuard
			}()
		}
	}
	worker.Wait()
	bar.FinishPrint("Processing Complete!")
}

func updateUser(u map[string]interface{}, espXmlmc *apiLib.XmlmcInstStruct) (bool, error) {
	buf2 := bytes.NewBufferString("")
	//-- Do we Lookup Site
	var p map[string]string
	p = make(map[string]string)
	for key, value := range u {
		p[key] = fmt.Sprintf("%s", value)
	}
	userID := p[AzureImportConf.AzureConf.UserID]
	for key := range userUpdateArray {
		field := userUpdateArray[key]
		value := AzureImportConf.UserMapping[field] //userMappingMap[name]

		t := template.New(field)
		t, _ = t.Parse(value)
		buf := bytes.NewBufferString("")
		t.Execute(buf, p)
		value = buf.String()
		if value == "%!s(<nil>)" {
			value = ""
		}

		//-- Process Site
		if field == "Site" {
			//-- Only use Site lookup if enabled and not set to Update only
			if AzureImportConf.SiteLookup.Enabled && AzureImportConf.OrgLookup.Action != updateString {
				value = getSiteFromLookup(value, buf2)
			}
		}

		//-- Skip UserType Field
		if field == "UserType" && !AzureImportConf.UpdateUserType {
			value = ""
		}

		//-- Skip Password Field
		if field == "Password" {
			value = ""
		}
		//-- if we have Value then set it
		if value != "" {
			espXmlmc.SetParam(field, value)
		}
	}

	//-- Check for Dry Run
	if configDryRun != true {
		XMLUpdate, xmlmcErr := espXmlmc.Invoke("admin", "userUpdate")
		var xmlRespon xmlmcResponse
		if xmlmcErr != nil {
			return false, xmlmcErr
		}
		err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			return false, err
		}

		if xmlRespon.MethodResult != constOK && xmlRespon.State.ErrorRet != noValuesToUpdate {
			err = errors.New(xmlRespon.State.ErrorRet)
			errorCountInc()
			return false, err

		}
		//-- Only use Org lookup if enabled and not set to create only
		if AzureImportConf.OrgLookup.Enabled && AzureImportConf.OrgLookup.Action != createString && len(AzureImportConf.OrgLookup.OrgUnits) > 0 {
			userAddGroups(p, buf2)
		}
		//-- Process User Status
		if AzureImportConf.UserAccountStatus.Enabled && AzureImportConf.UserAccountStatus.Action != createString {
			userSetStatus(userID, AzureImportConf.UserAccountStatus.Status, buf2)
		}

		//-- Add Roles
		if AzureImportConf.UserRoleAction != createString && len(AzureImportConf.Roles) > 0 {
			userAddRoles(userID, buf2, espXmlmc)
		}

		//-- Add Image
		if AzureImportConf.ImageLink.Enabled && AzureImportConf.ImageLink.Action != createString && AzureImportConf.ImageLink.URI != "" {
			userAddImage(p, buf2)
		}

		//-- Process Profile Details
		boolUpdateProfile := userUpdateProfile(p, buf2, espXmlmc)
		if boolUpdateProfile != true {
			err = errors.New("User Profile Issue (u): " + buf2.String())
			errorCountInc()
			return false, err
		}
		if xmlRespon.State.ErrorRet != noValuesToUpdate {
			buf2.WriteString(loggerGen(1, "User Update Success"))
			updateCountInc()
		} else {
			updateSkippedCountInc()
		}
		logger(1, buf2.String(), false)
		return true, nil
	}
	//-- Inc Counter
	updateSkippedCountInc()
	//-- DEBUG XML TO LOG FILE
	var XMLSTRING = espXmlmc.GetParam()
	logger(1, "User Update XML "+fmt.Sprintf("%s", XMLSTRING), false)
	espXmlmc.ClearParam()

	return true, nil
}

func userAddGroups(p map[string]string, buffer *bytes.Buffer) bool {
	for _, orgUnit := range AzureImportConf.OrgLookup.OrgUnits {
		userAddGroup(p, buffer, orgUnit)
	}
	return true
}
func userAddImage(p map[string]string, buffer *bytes.Buffer) {
	UserID := p[AzureImportConf.AzureConf.UserID]

	t := template.New("i" + UserID)
	t, _ = t.Parse(AzureImportConf.ImageLink.URI)
	buf := bytes.NewBufferString("")
	t.Execute(buf, p)
	value := buf.String()
	if value == "%!s(<nil>)" {
		value = ""
	}
	buffer.WriteString(loggerGen(2, "Image for user: "+value))
	if value == "" {
		return
	}

	if strings.ToUpper(AzureImportConf.ImageLink.UploadType) != "URI" {
		// get binary to upload via WEBDAV and then set value to relative "session" URI
		client := http.Client{
			Timeout: time.Duration(10 * time.Second),
		}

		rel_link := "session/" + UserID
		url := AzureImportConf.DAVURL + rel_link

		var imageB []byte
		var Berr error
		switch strings.ToUpper(AzureImportConf.ImageLink.UploadType) {
		case "URL":
			resp, err := http.Get(value)
			if err != nil {
				buffer.WriteString(loggerGen(4, "Unable to find "+value+" ["+fmt.Sprintf("%v", http.StatusInternalServerError)+"]"))
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 201 || resp.StatusCode == 200 {
				imageB, _ = ioutil.ReadAll(resp.Body)

			} else {
				buffer.WriteString(loggerGen(4, "Unsuccesful download: "+fmt.Sprintf("%v", resp.StatusCode)))
				return
			}
		default:
			imageB, Berr = hex.DecodeString(value[2:]) //stripping leading 0x
			if Berr != nil {
				buffer.WriteString(loggerGen(4, "Unsuccesful Decoding "+fmt.Sprintf("%v", Berr)))
				return
			}

		}
		//WebDAV upload
		if len(imageB) > 0 {
			putbody := bytes.NewReader(imageB)
			req, Perr := http.NewRequest("PUT", url, putbody)
			req.Header.Set("Content-Type", "image/jpeg")
			req.Header.Add("Authorization", "ESP-APIKEY "+AzureImportConf.APIKey)
			req.Header.Set("User-Agent", "Go-http-client/1.1")
			response, Perr := client.Do(req)
			if Perr != nil {
				buffer.WriteString(loggerGen(4, "PUT connection issue: "+fmt.Sprintf("%v", http.StatusInternalServerError)))
				return
			}
			defer response.Body.Close()
			_, _ = io.Copy(ioutil.Discard, response.Body)
			if response.StatusCode == 201 || response.StatusCode == 200 {
				buffer.WriteString(loggerGen(1, "Uploaded"))
				value = "/" + rel_link
			} else {
				buffer.WriteString(loggerGen(4, "Unsuccesful Upload: "+fmt.Sprintf("%v", response.StatusCode)))
				return
			}
		} else {
			buffer.WriteString(loggerGen(4, "No Image to upload"))
			return
		}
	}

	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)
	espXmlmc.SetParam("objectRef", "urn:sys:user:"+UserID)
	espXmlmc.SetParam("sourceImage", value)

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("activity", "profileImageSet")
	var xmlRespon xmlmcprofileSetImageResponse
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
		buffer.WriteString(loggerGen(4, "Unable to associate Image to User Profile: "+fmt.Sprintf("%v", xmlmcErr)))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Associate Image to User Profile: "+fmt.Sprintf("%v", err)))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Associate Image to User Profile: "+xmlRespon.State.ErrorRet))
		} else {
			buffer.WriteString(loggerGen(1, "Image added to User: "+UserID))
		}
	}
}
func userAddGroup(p map[string]string, buffer *bytes.Buffer, orgUnit OrgUnitStruct) bool {

	//-- Check if Site Attribute is set
	if orgUnit.Attribute == "" {
		buffer.WriteString(loggerGen(2, "Org Lookup is Enabled but Attribute is not Defined"))
		return false
	}
	//-- Get Value of Attribute
	t := template.New("orgunit" + strconv.Itoa(orgUnit.Type))
	t, _ = t.Parse(orgUnit.Attribute)
	buf := bytes.NewBufferString("")
	t.Execute(buf, p)
	value := buf.String()
	if value == "%!s(<nil>)" {
		value = ""
	}
	buffer.WriteString(loggerGen(2, "Azure Attribute for Org Lookup: "+value))
	if value == "" {
		return true
	}

	orgAttributeName := processComplexField(value)
	orgIsInCache, orgID := groupInCache(strconv.Itoa(orgUnit.Type) + orgAttributeName)
	//-- Check if we have Chached the site already
	if orgIsInCache {
		buffer.WriteString(loggerGen(1, "Found Org in Cache "+orgID))
		userAddGroupAsoc(p, orgUnit, orgID, buffer)
		return true
	}

	//-- We Get here if not in cache
	orgIsOnInstance, orgID := searchGroup(orgAttributeName, orgUnit, buffer)
	if orgIsOnInstance {
		buffer.WriteString(loggerGen(1, "Org Lookup found Id "+orgID))
		userAddGroupAsoc(p, orgUnit, orgID, buffer)
		return true
	}
	buffer.WriteString(loggerGen(1, "Unable to Find Organisation "+orgAttributeName))
	return false

}

func userAddGroupAsoc(p map[string]string, orgUnit OrgUnitStruct, orgID string, buffer *bytes.Buffer) {
	UserID := p[AzureImportConf.AzureConf.UserID]
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)
	espXmlmc.SetParam("userId", UserID)
	espXmlmc.SetParam("groupId", orgID)
	espXmlmc.SetParam("memberRole", orgUnit.Membership)
	espXmlmc.OpenElement("options")
	espXmlmc.SetParam("tasksView", strconv.FormatBool(orgUnit.TasksView))
	espXmlmc.SetParam("tasksAction", strconv.FormatBool(orgUnit.TasksAction))
	espXmlmc.CloseElement("options")

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("admin", "userAddGroup")
	var xmlRespon xmlmcuserSetGroupOptionsResponse
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
		buffer.WriteString(loggerGen(4, "Unable to Associate User To Group: "+fmt.Sprintf("%v", xmlmcErr)))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Associate User To Group: "+fmt.Sprintf("%v", err)))
	} else {
		if xmlRespon.MethodResult != constOK {
			if xmlRespon.State.ErrorRet != "The specified user ["+UserID+"] already belongs to ["+orgID+"] group" {
				buffer.WriteString(loggerGen(4, "Unable to Associate User To Organisation: "+xmlRespon.State.ErrorRet))
			} else {
				buffer.WriteString(loggerGen(1, "User: "+UserID+" Already Added to Organisation: "+orgID))
			}

		} else {
			buffer.WriteString(loggerGen(1, "User: "+UserID+" Added to Organisation: "+orgID))
		}
	}

}

//-- Function to Check if in Cache
func groupInCache(groupName string) (bool, string) {
	boolReturn := false
	stringReturn := ""
	//-- Check if in Cache
	mutexGroups.Lock()
	for _, group := range groups {
		if group.GroupName == groupName {
			boolReturn = true
			stringReturn = group.GroupID
			break
		}
	}
	mutexGroups.Unlock()
	return boolReturn, stringReturn
}

//-- Function to Check if site is on the instance
func searchGroup(orgName string, orgUnit OrgUnitStruct, buffer *bytes.Buffer) (bool, string) {
	boolReturn := false
	strReturn := ""
	//-- ESP Query for site
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)
	if orgName == "" {
		return boolReturn, strReturn
	}
	espXmlmc.SetParam("application", "com.hornbill.core")
	espXmlmc.SetParam("queryName", "GetGroupByName")
	espXmlmc.OpenElement("queryParams")
	espXmlmc.SetParam("h_name", orgName)
	espXmlmc.SetParam("h_type", strconv.Itoa(orgUnit.Type))
	espXmlmc.CloseElement("queryParams")

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
	var xmlRespon xmlmcGroupListResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+fmt.Sprintf("%v", xmlmcErr)))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+fmt.Sprintf("%v", err)))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.GroupID != "" {
				strReturn = xmlRespon.Params.RowData.Row.GroupID
				boolReturn = true
				//-- Add Group to Cache
				mutexGroups.Lock()
				var newgroupForCache groupListStruct
				newgroupForCache.GroupID = strReturn
				newgroupForCache.GroupName = strconv.Itoa(orgUnit.Type) + orgName
				name := []groupListStruct{newgroupForCache}
				groups = append(groups, name...)
				mutexGroups.Unlock()
			}
		}
	}

	return boolReturn, strReturn
}

func createUser(u map[string]interface{}, espXmlmc *apiLib.XmlmcInstStruct) (bool, error) {
	buf2 := bytes.NewBufferString("")
	//-- Do we Lookup Site
	var p map[string]string
	p = make(map[string]string)

	for key, value := range u {
		p[key] = fmt.Sprintf("%s", value)
	}

	userID := p[AzureImportConf.AzureConf.UserID]

	//-- Loop Through UserProfileMapping
	for key := range userCreateArray {
		field := userCreateArray[key]
		value := AzureImportConf.UserMapping[field] //userMappingMap[name]
		t := template.New(field)
		t, _ = t.Parse(value)
		buf := bytes.NewBufferString("")
		t.Execute(buf, p)
		value = buf.String()
		if value == "%!s(<nil>)" {
			value = ""
		}

		//-- Process Site
		if field == "Site" {
			//-- Only use Site lookup if enabled and not set to Update only
			if AzureImportConf.SiteLookup.Enabled && AzureImportConf.OrgLookup.Action != updateString {
				value = getSiteFromLookup(value, buf2)
			}
		}
		//-- Process Password Field
		if field == "Password" {
			if value == "" {
				value = generatePasswordString(10)
				logger(1, "Auto Generated Password for: "+userID+" - "+value, false)
			}
			value = base64.StdEncoding.EncodeToString([]byte(value))
		}

		//-- if we have Value then set it
		if value != "" {
			espXmlmc.SetParam(field, value)

		}
	}

	//-- Check for Dry Run
	if configDryRun != true {
		XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userCreate")
		var xmlRespon xmlmcResponse
		if xmlmcErr != nil {
			errorCountInc()
			return false, xmlmcErr
		}
		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			errorCountInc()
			return false, err
		}
		if xmlRespon.MethodResult != constOK {
			err = errors.New(xmlRespon.State.ErrorRet)
			errorCountInc()
			return false, err

		}
		logger(1, "User Create Success", false)

		//-- Only use Org lookup if enabled and not set to Update only
		if AzureImportConf.OrgLookup.Enabled && AzureImportConf.OrgLookup.Action != updateString && len(AzureImportConf.OrgLookup.OrgUnits) > 0 {
			userAddGroups(p, buf2)
		}
		//-- Process Account Status
		if AzureImportConf.UserAccountStatus.Enabled && AzureImportConf.UserAccountStatus.Action != updateString {
			userSetStatus(userID, AzureImportConf.UserAccountStatus.Status, buf2)
		}

		if AzureImportConf.UserRoleAction != updateString && len(AzureImportConf.Roles) > 0 {
			userAddRoles(userID, buf2, espXmlmc)
		}

		//-- Add Image
		if AzureImportConf.ImageLink.Enabled && AzureImportConf.ImageLink.Action != updateString && AzureImportConf.ImageLink.URI != "" {
			userAddImage(p, buf2)
		}

		//-- Process Profile Details
		boolUpdateProfile := userUpdateProfile(p, buf2, espXmlmc)
		if boolUpdateProfile != true {
			err = errors.New("User Profile issue (c): " + buf2.String())
			errorCountInc()
			return false, err
		}

		logger(1, buf2.String(), false)
		createCountInc()
		return true, nil
	} else {
		//-- Process Profile Details as part of the dry run for testing
	}
	//-- DEBUG XML TO LOG FILE
	var XMLSTRING = espXmlmc.GetParam()
	logger(1, "User Create XML "+fmt.Sprintf("%s", XMLSTRING), false)
	createSkippedCountInc()
	espXmlmc.ClearParam()

	return true, nil
}

func userUpdateProfile(p map[string]string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) bool {
	UserID := p[AzureImportConf.AzureConf.UserID]
	buffer.WriteString(loggerGen(1, "Processing User Profile Data "+UserID))
	espXmlmc.OpenElement("profileData")
	espXmlmc.SetParam("userID", UserID)
	//-- Loop Through UserProfileMapping
	for key := range userProfileArray {
		field := userProfileArray[key]
		value := AzureImportConf.UserProfileMapping[field]

		t := template.New(field)
		t, _ = t.Parse(value)
		buf := bytes.NewBufferString("")
		t.Execute(buf, p)
		value = buf.String()
		if value == "%!s(<nil>)" {
			value = ""
		}

		if field == "Manager" {
			//-- Process User manager
			if AzureImportConf.UserManagerMapping.Enabled && AzureImportConf.UserManagerMapping.Action != updateString {
				value = getManagerFromLookup(value, buffer)
			}
		}

		//-- if we have Value then set it
		if value != "" {
			espXmlmc.SetParam(field, value)
		}
	}

	espXmlmc.CloseElement("profileData")
	//-- Check for Dry Run
	if configDryRun != true {
		XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userProfileSet")
		var xmlRespon xmlmcResponse
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "Unable to Update User Profile: "+fmt.Sprintf("%v", xmlmcErr)))
			return false
		}
		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Unable to Update User Profile: "+fmt.Sprintf("%v", err)))

			return false
		}
		if xmlRespon.MethodResult != constOK {
			profileSkippedCountInc()
			if xmlRespon.State.ErrorRet == noValuesToUpdate {
				return true
			}
			err := errors.New(xmlRespon.State.ErrorRet)
			buffer.WriteString(loggerGen(4, "Unable to Update User Profile: "+fmt.Sprintf("%v", err)))
			return false
		}
		profileCountInc()
		buffer.WriteString(loggerGen(1, "User Profile Update Success"))
		return true

	}
	//-- DEBUG XML TO LOG FILE
	var XMLSTRING = espXmlmc.GetParam()
	buffer.WriteString(loggerGen(1, "User Profile Update XML "+fmt.Sprintf("%s", XMLSTRING)))
	profileSkippedCountInc()
	espXmlmc.ClearParam()
	return true

}

func userSetStatus(userID string, status string, buffer *bytes.Buffer) bool {
	buffer.WriteString(loggerGen(1, "Set Status for User: "+fmt.Sprintf("%s", userID)+" Status:"+fmt.Sprintf("%s", status)))

	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)

	espXmlmc.SetParam("userId", userID)
	espXmlmc.SetParam("accountStatus", status)

	XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userSetAccountStatus")

	var XMLSTRING = espXmlmc.GetParam()
	buffer.WriteString(loggerGen(1, "User Create XML "+fmt.Sprintf("%s", XMLSTRING)))

	var xmlRespon xmlmcResponse
	if xmlmcErr != nil {
		logger(4, "Unable to Set User Status: "+fmt.Sprintf("%s", xmlmcErr), true)

	}
	err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Set User Status "+fmt.Sprintf("%s", err)))
		return false
	}
	if xmlRespon.MethodResult != constOK {
		if xmlRespon.State.ErrorRet != "Failed to update account status (target and the current status is the same)." {
			buffer.WriteString(loggerGen(4, "Unable to Set User Status 111: "+xmlRespon.State.ErrorRet))
			return false
		}
		buffer.WriteString(loggerGen(1, "User Status Already Set to: "+fmt.Sprintf("%s", status)))
		return true
	}
	buffer.WriteString(loggerGen(1, "User Status Set Successfully"))
	return true
}

func userAddRoles(userID string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) bool {

	espXmlmc.SetParam("userId", userID)
	for _, role := range AzureImportConf.Roles {
		espXmlmc.SetParam("role", role)
		buffer.WriteString(loggerGen(1, "Add Role to User: "+role))
	}
	XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userAddRole")
	var xmlRespon xmlmcResponse
	if xmlmcErr != nil {
		logger(4, "Unable to Assign Role to User: "+fmt.Sprintf("%s", xmlmcErr), true)

	}
	err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Assign Role to User: "+fmt.Sprintf("%s", err)))
		return false
	}
	if xmlRespon.MethodResult != constOK {
		buffer.WriteString(loggerGen(4, "Unable to Assign Role to User: "+xmlRespon.State.ErrorRet))
		return false
	}
	buffer.WriteString(loggerGen(1, "Roles Added Successfully"))
	return true
}

func checkUserOnInstance(userID string, espXmlmc *apiLib.XmlmcInstStruct) (bool, error) {

	espXmlmc.SetParam("entity", "UserAccount")
	espXmlmc.SetParam("keyValue", userID)
	XMLCheckUser, xmlmcErr := espXmlmc.Invoke("data", "entityDoesRecordExist")
	var xmlRespon xmlmcCheckUserResponse
	if xmlmcErr != nil {
		return false, xmlmcErr
	}
	err := xml.Unmarshal([]byte(XMLCheckUser), &xmlRespon)
	if err != nil {
		stringError := err.Error()
		stringBody := string(XMLCheckUser)
		errWithBody := errors.New(stringError + " RESPONSE BODY: " + stringBody)
		return false, errWithBody
	}
	if xmlRespon.MethodResult != constOK {
		err := errors.New(xmlRespon.State.ErrorRet)
		return false, err
	}
	return xmlRespon.Params.RecordExist, nil
}

//-- Function to search for site
func getSiteFromLookup(site string, buffer *bytes.Buffer) string {
	siteReturn := ""

	//-- Get Value of Attribute
	siteAttributeName := processComplexField(site)
	buffer.WriteString(loggerGen(1, "Looking Up Site: "+siteAttributeName))
	if siteAttributeName == "" {
		return ""
	}
	siteIsInCache, SiteIDCache := siteInCache(siteAttributeName)
	//-- Check if we have Cached the site already
	if siteIsInCache {
		siteReturn = strconv.Itoa(SiteIDCache)
		buffer.WriteString(loggerGen(1, "Found Site in Cache: "+siteReturn))
	} else {
		siteIsOnInstance, SiteIDInstance := searchSite(siteAttributeName, buffer)
		//-- If Returned set output
		if siteIsOnInstance {
			siteReturn = strconv.Itoa(SiteIDInstance)
		}
	}
	buffer.WriteString(loggerGen(1, "Site Lookup found ID: "+siteReturn))
	return siteReturn
}

func processComplexField(s string) string {
	return html.UnescapeString(s)
}

//-- Function to Check if in Cache
func siteInCache(siteName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	mutexSites.Lock()
	//-- Check if in Cache
	for _, site := range sites {
		if site.SiteName == siteName {
			boolReturn = true
			intReturn = site.SiteID
			break
		}
	}
	mutexSites.Unlock()
	return boolReturn, intReturn
}

//-- Function to Check if site is on the instance
func searchSite(siteName string, buffer *bytes.Buffer) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for site
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)
	if siteName == "" {
		return boolReturn, intReturn
	}
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")

	var xmlRespon xmlmcSiteListResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+fmt.Sprintf("%v", xmlmcErr)))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+fmt.Sprintf("%v", err)))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.SiteName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.SiteName) == strings.ToLower(siteName) {
					intReturn = xmlRespon.Params.RowData.Row.SiteID
					boolReturn = true
					//-- Add Site to Cache
					mutexSites.Lock()
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					name := []siteListStruct{newSiteForCache}
					sites = append(sites, name...)
					mutexSites.Unlock()
				}
			}
		}
	}

	return boolReturn, intReturn
}

func getManagerFromLookup(manager string, buffer *bytes.Buffer) string {

	if manager == "" {
		buffer.WriteString(loggerGen(1, "No Manager to search"))
		return ""
	}
	//-- Get Value of Attribute
	ManagerAttributeName := processComplexField(manager)
	buffer.WriteString(loggerGen(1, "Manager Lookup: "+ManagerAttributeName))

	//-- Dont Continue if we didn't get anything
	if ManagerAttributeName == "" {
		return ""
	}

	buffer.WriteString(loggerGen(1, "Looking Up Manager "+ManagerAttributeName))
	managerIsInCache, ManagerIDCache := managerInCache(ManagerAttributeName)

	//-- Check if we have Chached the site already
	if managerIsInCache {
		buffer.WriteString(loggerGen(1, "Found Manager in Cache "+ManagerIDCache))
		return ManagerIDCache
	}
	buffer.WriteString(loggerGen(1, "Manager Not In Cache Searching"))
	ManagerIsOnInstance, ManagerIDInstance := searchManager(ManagerAttributeName, buffer)
	//-- If Returned set output
	if ManagerIsOnInstance {
		buffer.WriteString(loggerGen(1, "Manager Lookup found Id "+ManagerIDInstance))

		return ManagerIDInstance
	}

	return ""
}

//-- Search Manager on Instance
func searchManager(managerName string, buffer *bytes.Buffer) (bool, string) {
	boolReturn := false
	strReturn := ""
	//-- ESP Query for site
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)
	espXmlmc.SetTrace("AzureUserImport")
	if managerName == "" {
		return boolReturn, strReturn
	}

	espXmlmc.SetParam("entity", "UserAccount")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", managerName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLUserSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	var xmlRespon xmlmcUserListResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Manager: "+fmt.Sprintf("%v", xmlmcErr)))
	}
	err := xml.Unmarshal([]byte(XMLUserSearch), &xmlRespon)
	if err != nil {
		stringError := err.Error()
		stringBody := string(XMLUserSearch)
		buffer.WriteString(loggerGen(4, "Unable to Search for Manager: "+fmt.Sprintf("%v", stringError+" RESPONSE BODY: "+stringBody)))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Search for Manager: "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.UserName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.UserName) == strings.ToLower(managerName) {

					strReturn = xmlRespon.Params.RowData.Row.UserID
					boolReturn = true
					//-- Add Site to Cache
					mutexManagers.Lock()
					var newManagerForCache managerListStruct
					newManagerForCache.UserID = strReturn
					newManagerForCache.UserName = managerName
					name := []managerListStruct{newManagerForCache}
					managers = append(managers, name...)
					mutexManagers.Unlock()
				}
			}
		}
	}
	return boolReturn, strReturn
}

//-- Check if Manager in Cache
func managerInCache(managerName string) (bool, string) {
	boolReturn := false
	stringReturn := ""
	//-- Check if in Cache
	mutexManagers.Lock()
	for _, manager := range managers {
		if strings.ToLower(manager.UserName) == strings.ToLower(managerName) {
			boolReturn = true
			stringReturn = manager.UserID
		}
	}
	mutexManagers.Unlock()
	return boolReturn, stringReturn
}

//-- Generate Password String
func generatePasswordString(n int) string {
	var arbytes = make([]byte, n)
	rand.Read(arbytes)
	for i, b := range arbytes {
		arbytes[i] = letterBytes[b%byte(len(letterBytes))]
	}
	return string(arbytes)
}

// =================== COUNTERS =================== //
func errorCountInc() {
	mutexCounters.Lock()
	errorCount++
	mutexCounters.Unlock()
}
func updateCountInc() {
	mutexCounters.Lock()
	counters.updated++
	mutexCounters.Unlock()
}
func updateSkippedCountInc() {
	mutexCounters.Lock()
	counters.updatedSkipped++
	mutexCounters.Unlock()
}
func createSkippedCountInc() {
	mutexCounters.Lock()
	counters.createskipped++
	mutexCounters.Unlock()
}
func createCountInc() {
	mutexCounters.Lock()
	counters.created++
	mutexCounters.Unlock()
}
func profileCountInc() {
	mutexCounters.Lock()
	counters.profileUpdated++
	mutexCounters.Unlock()
}
func profileSkippedCountInc() {
	mutexCounters.Lock()
	counters.profileSkipped++
	mutexCounters.Unlock()
}
