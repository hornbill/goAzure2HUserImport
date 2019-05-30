package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

//-- Function to search for site
func getSiteFromLookup(site string, buffer *bytes.Buffer) string {
	siteReturn := ""

	//-- Get Value of Attribute
	siteAttributeName := processComplexField(site)
	buffer.WriteString(loggerGen(1, "Looking Up Site: "+siteAttributeName, true))
	if siteAttributeName == "" {
		return ""
	}
	siteIsInCache, SiteIDCache := siteInCache(siteAttributeName)
	//-- Check if we have Cached the site already
	if siteIsInCache {
		siteReturn = strconv.Itoa(SiteIDCache)
		buffer.WriteString(loggerGen(1, "Found Site in Cache: "+siteReturn, true))
	} else {
		siteIsOnInstance, SiteIDInstance := searchSite(siteAttributeName, buffer)
		//-- If Returned set output
		if siteIsOnInstance {
			siteReturn = strconv.Itoa(SiteIDInstance)
		}
	}
	buffer.WriteString(loggerGen(1, "Site Lookup found ID: "+siteReturn, true))
	return siteReturn
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
	if siteName == "" {
		return boolReturn, intReturn
	}
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.InstanceID)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("column", "h_site_name")
	espXmlmc.SetParam("value", siteName)
	//espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")

	var xmlRespon xmlmcSiteListResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+fmt.Sprintf("%v", xmlmcErr), true))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+fmt.Sprintf("%v", err), true))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, true))
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.SiteName != "" {
				if strings.EqualFold(xmlRespon.Params.RowData.Row.SiteName, siteName) {
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
