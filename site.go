package main

import (
	"strings"
)

//-- Function to search for site
func getSiteFromLookup(importData *userWorkingDataStruct) string {
	//-- Check if Site Attribute is set
	if AzureImportConf.User.Site.Value == "" {
		logger(4, "Site Lookup is Enabled but Attribute is not Defined", false)
		return ""
	}
	//-- Get Value of Attribute
	logger(1, "DB Attribute for Site Lookup: "+AzureImportConf.User.Site.Value, false)

	//-- Get Value of Attribute
	siteAttributeName := processComplexField(importData.DB, AzureImportConf.User.Site.Value)
	siteAttributeName = processImportAction(importData.Custom, siteAttributeName)
	logger(1, "Looking Up Site "+siteAttributeName, false)
	siteIsInCache, SiteIDCache := siteInCache(siteAttributeName)
	//-- Check if we have Chached the site already
	if siteIsInCache {
		logger(1, "Found Site in Cache"+SiteIDCache, false)
		return SiteIDCache
	}

	logger(1, "Unable to Locate Site", false)
	return ""
}

//-- Function to Check if in Cache
func siteInCache(siteName string) (bool, string) {
	//-- Check if in Cache
	_, found := HornbillCache.Sites[strings.ToLower(siteName)]
	if found {
		return true, HornbillCache.Sites[strings.ToLower(siteName)].HID
	}
	return false, ""
}
