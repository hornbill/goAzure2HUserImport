package main

import (
	"sync"
	"time"
)

const (
	version      = "2.1.1"
	constOK      = "ok"
	updateString = "Update"
	createString = "Create"
	apiResource  = "https://graph.microsoft.com"
)

//Password profiles
var passwordProfile passwordProfileStruct
var blacklistURLs = [...]string{"https://files.hornbill.com/hornbillStatic/password_blacklists/SplashData.txt", "https://files.hornbill.com/hornbillStatic/password_blacklists/Imperva.txt"}
var defaultPasswordLength = 16

type passwordProfileStruct struct {
	Length              int
	UseLower            bool
	ForceLower          int
	UseUpper            bool
	ForceUpper          int
	UseNumeric          bool
	ForceNumeric        int
	UseSpecial          bool
	ForceSpecial        int
	Blacklist           []string
	CheckMustNotContain bool
}

type xmlmcSettingResponse struct {
	Params struct {
		Option []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"option"`
	} `json:"params"`
	State stateStruct `json:"state"`
}

var (
	//AzureImportConf - Holds the import configuration from the JSON file
	AzureImportConf    AzureImportConfStruct
	sites              []siteListStruct
	managers           []managerListStruct
	groups             []groupListStruct
	counters           counterTypeStruct
	configFileName     string
	configZone         string
	configLogPrefix    string
	configDryRun       bool
	configVersion      bool
	configMaxRoutines  int
	timeNow            string
	strAzurePagerToken = ""
	startTime          time.Time
	endTime            time.Duration
	errorCount         uint64
	noValuesToUpdate   = "There are no values to update"
	mutexBar           = &sync.Mutex{}
	mutexCounters      = &sync.Mutex{}
	mutexSites         = &sync.Mutex{}
	mutexGroups        = &sync.Mutex{}
	mutexManagers      = &sync.Mutex{}
	logFileMutex       = &sync.Mutex{}
	worker             sync.WaitGroup
	globalBearerToken  = ""
	globalTokenExpiry  int64

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

type counterTypeStruct struct {
	updated        uint16
	created        uint16
	profileUpdated uint16
	updatedSkipped uint16
	createskipped  uint16
	profileSkipped uint16
}

type userAccountStatusStruct struct {
	Action  string
	Enabled bool
	Status  string
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

type azureConfStruct struct {
	Tenant         string
	ClientID       string
	ClientSecret   string
	UserFilter     string
	UserProperties []string
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
