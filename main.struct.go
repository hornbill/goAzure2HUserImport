package main

import (
	"sync"
	"time"
)

const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	version      = "1.4.4"
	constOK      = "ok"
	updateString = "Update"
	createString = "Create"
)

var (
	//AzureImportConf - Holds the import configuration from the JSON file
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
	strAzurePagerToken  = ""
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
	globalBearerToken   = ""
	globalTokenExpiry   int64

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

//OrgUnitStruct - defines Organisational Unit struct
type OrgUnitStruct struct {
	Attribute   string
	Type        int
	Membership  string
	TasksView   bool
	TasksAction bool
}

//AzureImportConfStruct - defines JSON configuration output
type AzureImportConfStruct struct {
	APIKey             string
	InstanceID         string
	URL                string
	DAVURL             string
	UpdateUserType     bool
	UserRoleAction     string
	UserIdentifier     string
	AzureConf          azureConfStruct
	UserMapping        map[string]string
	UserAccountStatus  userAccountStatusStruct
	UserProfileMapping map[string]string
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

type azureConfStruct struct {
	Tenant         string
	ClientID       string
	ClientSecret   string
	UserFilter     string
	UserID         string
	Debug          bool
	APIVersion     string
	Search         string
	UsersByGroupID []groupConfStruct
}

type groupConfStruct struct {
	ObjectID string
	Name     string
}

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
