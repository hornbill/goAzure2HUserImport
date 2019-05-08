package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

//processUsers -- Processes Users from User Map
//--If user already exists on the instance, update
//--If user doesn't exist, create
func processUsers(arrUsers []map[string]interface{}) {
	bar := pb.StartNew(len(arrUsers))
	logger(1, "Processing Users", false)

	//Get the identity of the UserID field from the config
	userIDField := fmt.Sprintf("%v", AzureImportConf.AzureConf.UserID)
	//-- Loop each user
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)

	for _, customerRecord := range arrUsers {
		maxGoroutinesGuard <- struct{}{}
		worker.Add(1)
		userMap := customerRecord
		//Get the user ID for the current record
		userID := fmt.Sprintf("%v", userMap[userIDField])
		logger(1, "User ID: "+userID, false)
		if userID != "" {
			//logger(1, "User ID: "+fmt.Sprintf("%v", userID), false)
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
				//-- Update or Create User
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
		p[key] = fmt.Sprintf("%v", value)
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
		if value != "" && value != "<nil>" && value != "&lt;nil&gt;" {
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
	logger(1, "User Update XML "+fmt.Sprintf("%v", XMLSTRING), false)
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

	strContentType := "image/jpeg"

	if strings.ToUpper(AzureImportConf.ImageLink.UploadType) != "URI" {
		// get binary to upload via WEBDAV and then set value to relative "session" URI
		client := http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Timeout: time.Duration(10 * time.Second),
		}

		relLink := "session/" + UserID
		strDAVurl := AzureImportConf.DAVURL + relLink

		var imageB []byte
		var Berr error
		switch strings.ToUpper(AzureImportConf.ImageLink.UploadType) {
		case "AZURE":

			strBearerToken, err := getBearerToken()
			if err != nil || strBearerToken == "" {
				logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
				return
			}

			strTenant := AzureImportConf.AzureConf.Tenant
			strURL := "https://graph.windows.net/" + strTenant + "/users/" + strings.Replace(UserID, "@", "%40", -1) + "/thumbnailPhoto?"

			data := url.Values{}
			data.Set("api-version", AzureImportConf.AzureConf.APIVersion)
			strData := data.Encode()
			strURL += strData
			req, err := http.NewRequest("GET", strURL, nil) //, bytes.NewBuffer(""))
			req.Header.Set("User-Agent", "Go-http-client/1.1")
			req.Header.Set("Authorization", "Bearer "+strBearerToken)
			duration := time.Second * time.Duration(30)
			imgclient := &http.Client{Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			}, Timeout: duration}
			resp, err := imgclient.Do(req)
			if err != nil {
				logger(4, " [Image] Connection Error: "+fmt.Sprintf("%v", err), false)
				return
			}
			defer resp.Body.Close()

			//-- Check for HTTP Response
			if resp.StatusCode != 200 {
				if resp.StatusCode != 404 {
					errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
					err = errors.New(errorString)
					logger(4, " [Image] Error: "+fmt.Sprintf("%v", err), false)
				} else {
					logger(4, " [Image] Not Found", false)
				}
				//Drain the body so we can reuse the connection
				io.Copy(ioutil.Discard, resp.Body)
				return
			}
			logger(2, "[Image] Connection Successful", false)

			imageB, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				logger(4, " [Image] Cannot read the body of the response", false)
				return
			}
			strContentType = resp.Header.Get("Content-Type")

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
			req, Perr := http.NewRequest("PUT", strDAVurl, putbody)
			req.Header.Set("Content-Type", strContentType)
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
				value = "/" + relLink
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
		p[key] = fmt.Sprintf("%v", value)
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
		if value != "" && value != "<nil>" && value != "&lt;nil&gt;" {
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
	}

	//-- DEBUG XML TO LOG FILE
	var XMLSTRING = espXmlmc.GetParam()
	logger(1, "User Create XML "+fmt.Sprintf("%v", XMLSTRING), false)
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
		if value != "" && value != "<nil>" && value != "&lt;nil&gt;" {
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
	buffer.WriteString(loggerGen(1, "User Profile Update XML "+fmt.Sprintf("%v", XMLSTRING)))
	profileSkippedCountInc()
	espXmlmc.ClearParam()
	return true

}

func userSetStatus(userID string, status string, buffer *bytes.Buffer) bool {
	buffer.WriteString(loggerGen(1, "Set Status for User: "+fmt.Sprintf("%v", userID)+" Status:"+fmt.Sprintf("%v", status)))

	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.URL)
	espXmlmc.SetAPIKey(AzureImportConf.APIKey)

	espXmlmc.SetParam("userId", userID)
	espXmlmc.SetParam("accountStatus", status)

	XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userSetAccountStatus")

	var XMLSTRING = espXmlmc.GetParam()
	buffer.WriteString(loggerGen(1, "User Create XML "+fmt.Sprintf("%v", XMLSTRING)))

	var xmlRespon xmlmcResponse
	if xmlmcErr != nil {
		logger(4, "Unable to Set User Status: "+fmt.Sprintf("%v", xmlmcErr), true)

	}
	err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Set User Status "+fmt.Sprintf("%v", err)))
		return false
	}
	if xmlRespon.MethodResult != constOK {
		if xmlRespon.State.ErrorRet != "Failed to update account status (target and the current status is the same)." {
			buffer.WriteString(loggerGen(4, "Unable to Set User Status ("+fmt.Sprintf("%v", status)+"): "+xmlRespon.State.ErrorRet))
			return false
		}
		buffer.WriteString(loggerGen(1, "User Status Already Set to: "+fmt.Sprintf("%v", status)))
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
		logger(4, "Unable to Assign Role to User: "+fmt.Sprintf("%v", xmlmcErr), true)

	}
	err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Assign Role to User: "+fmt.Sprintf("%v", err)))
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
	espXmlmc.SetParam("column", "h_site_name")
	espXmlmc.SetParam("value", siteName)
	//espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")

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
	espXmlmc.SetParam("column", "h_user_id")
	espXmlmc.SetParam("value", managerName)
	//espXmlmc.SetParam("h_user_id", managerName)
	//	espXmlmc.SetParam("h_name", managerName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLUserSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
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
				if strings.ToLower(xmlRespon.Params.RowData.Row.UserID) == strings.ToLower(managerName) {

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
