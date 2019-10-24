package main

import (
	"fmt"
	"regexp"
	"strings"
)

func getManager(importData *userWorkingDataStruct, currentData userAccountStruct) string {
	//-- Check if Manager Attribute is set
	if AzureImportConf.User.Manager.Value == "" {
		logger(4, "Manager Lookup is Enabled but Attribute is not Defined", false)
		return ""
	}

	//-- Get Value of Attribute
	logger(1, "DB Attribute for Manager Lookup: "+AzureImportConf.User.Manager.Value, false)

	//-- Get Value of Attribute
	ManagerAttributeName := processComplexField(importData.DB, AzureImportConf.User.Manager.Value)
	ManagerAttributeName = processImportAction(importData.Custom, ManagerAttributeName)

	if AzureImportConf.User.Manager.Options.MatchAgainstDistinguishedName {
		logger(1, "Searching Distinguished Name Cache for: "+ManagerAttributeName, false)
		managerID := getUserFromDNCache(ManagerAttributeName)
		if managerID != "" {
			logger(1, "Found Manager in Distinguished Name  Cache: "+managerID, false)
			return managerID
		}
		logger(1, "Unable to find Manager in Distinguished Name  Cache Coninuing search", false)
	}

	//-- Dont Continue if we didn't get anything
	if ManagerAttributeName == "" {
		return ""
	}

	//-- Pull Data from Attriute using regext
	if AzureImportConf.User.Manager.Options.GetStringFromValue.Regex != "" {
		logger(1, "DB Manager String: "+ManagerAttributeName, false)
		ManagerAttributeName = getNameFromDBString(ManagerAttributeName)
	}
	//-- Is Search Enabled
	if AzureImportConf.User.Manager.Options.Search.Enable {
		logger(1, "Search for Manager is Enabled", false)

		logger(1, "Looking Up Manager from Cache: "+ManagerAttributeName, false)
		managerIsInCache, ManagerIDCache := managerInCache(ManagerAttributeName)

		//-- Check if we have Chached the site already
		if managerIsInCache {
			logger(1, "Found Manager in Cache: "+ManagerIDCache, false)
			return ManagerIDCache
		}
		logger(1, "Manager Not In Cache Searching Hornbill Data", false)
		ManagerIsOnInstance, ManagerIDInstance := searchManager(ManagerAttributeName)
		//-- If Returned set output
		if ManagerIsOnInstance {
			logger(1, "Manager Lookup found Id: "+ManagerIDInstance, false)
			return ManagerIDInstance
		}
	} else {
		logger(1, "Search for Manager is Disabled", false)
		//-- Assume data is manager id
		logger(1, "Manager Id: "+ManagerAttributeName, false)
		return ManagerAttributeName
	}

	//else return empty
	return ""
}

//-- Search Manager on Instance
func searchManager(managerName string) (bool, string) {
	//-- ESP Query for site
	if managerName == "" {
		return false, ""
	}

	//-- Add support for Search Feild configuration
	strSearchField := "h_name"
	if AzureImportConf.User.Manager.Options.Search.SearchField != "" {
		strSearchField = AzureImportConf.User.Manager.Options.Search.SearchField
	}

	logger(1, "Manager Search: "+strSearchField+" - "+managerName, false)

	//-- Check User Cache for Manager
	for _, v := range HornbillCache.Users {
		if strings.EqualFold(v.HName, managerName) {
			//-- If not already in cache push to cache
			_, found := HornbillCache.Managers[strings.ToLower(managerName)]
			if !found {
				HornbillCache.Managers[strings.ToLower(managerName)] = v.HUserID
			}
			return true, v.HUserID
		}
	}

	return false, ""
}

//-- Check if Manager in Cache
func managerInCache(managerName string) (bool, string) {
	//-- Check if in Cache
	_, found := HornbillCache.Managers[strings.ToLower(managerName)]
	if found {
		return true, HornbillCache.Managers[strings.ToLower(managerName)]
	}
	return false, ""
}

//-- Takes a string based on a DB DN and returns to the CN String Name
func getNameFromDBString(feild string) string {

	regex := AzureImportConf.User.Manager.Options.GetStringFromValue.Regex
	reverse := AzureImportConf.User.Manager.Options.GetStringFromValue.Reverse
	stringReturn := ""

	//-- Match $variables from String
	re1, err := regexp.Compile(regex)
	if err != nil {
		logger(4, "Error Compiling Regex: "+regex+" Error: "+fmt.Sprintf("%v", err), false)

	}
	//-- Get Array of all Matched max 100
	result := re1.FindAllString(feild, 100)

	//-- Loop Matches
	for _, v := range result {
		//-- String DB String Chars Out from match
		v = strings.Replace(v, "CN=", "", -1)
		v = strings.Replace(v, "OU=", "", -1)
		v = strings.Replace(v, "DC=", "", -1)
		v = strings.Replace(v, "\\", "", -1)
		nameArray := strings.Split(v, ",")

		for _, n := range nameArray {
			n = strings.Trim(n, " ")
			if n != "" {
				if reverse {
					stringReturn = n + " " + stringReturn
				} else {
					stringReturn = stringReturn + " " + n
				}
			}

		}

	}
	stringReturn = strings.Trim(stringReturn, " ")
	return stringReturn
}
