package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"

	apiLib "github.com/hornbill/goApiLib"
)

//groupInCache - checks if group in local cache
func groupInCache(groupName string) (bool, string) {
	boolReturn := false
	stringReturn := ""
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

// searchGroup - checks if Group is on the instance
func searchGroup(orgName string, orgUnit OrgUnitStruct, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
	boolReturn := false
	strReturn := ""
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
		buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+fmt.Sprintf("%v", xmlmcErr), false))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+fmt.Sprintf("%v", err), false))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Search for Group: "+xmlRespon.State.ErrorRet, false))
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
