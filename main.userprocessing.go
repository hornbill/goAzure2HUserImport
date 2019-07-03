package main

import (
	"bytes"
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

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

//processUsers -- Processes Users from User Map
//--If user already exists on the instance, update
//--If user doesn't exist, create
func processUsers(arrUsers []map[string]interface{}) {
	bar := pb.StartNew(len(arrUsers))
	logger(1, "Processing Users", false)
	logger(0, "", false)

	//Get the identity of the UserID field from the config
	userIDField := fmt.Sprintf("%v", AzureImportConf.AzureConf.UserID)

	//-- Loop each user
	maxGoroutinesGuard := make(chan struct{}, configMaxRoutines)

	for _, customerRecord := range arrUsers {
		maxGoroutinesGuard <- struct{}{}
		worker.Add(1)
		userMap := customerRecord
		//Get the user ID for the current record
		userID := fmt.Sprintf("%v", userMap[userIDField])
		if userID != "" {
			espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.InstanceID)
			espXmlmc.SetAPIKey(AzureImportConf.APIKey)
			go func() {
				defer worker.Done()

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
						updateUser(userMap, espXmlmc)
					} else {
						createUser(userMap, espXmlmc)
					}
				}
				<-maxGoroutinesGuard
			}()
		}
	}
	worker.Wait()
	bar.FinishPrint("Processing Complete!")
}

func createUser(u map[string]interface{}, espXmlmc *apiLib.XmlmcInstStruct) {
	buf2 := bytes.NewBufferString("")
	p := make(map[string]string)
	for key, value := range u {
		p[key] = fmt.Sprintf("%v", value)
	}

	userID := p[AzureImportConf.AzureConf.UserID]
	userFirstName := p["givenName"]
	userLastName := p["surname"]

	buf2.WriteString(loggerGen(1, "Create User: "+userID, false))
	for key := range userCreateArray {
		field := userCreateArray[key]
		value := AzureImportConf.UserMapping[field]

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
			if configDebug {
				buf2.WriteString(loggerGen(1, "[PASSWORD] Key Exists. Value: ["+value+"]", true))
			}
			if value == "" {
				if configDebug {
					buf2.WriteString(loggerGen(1, "[PASSWORD] Value empty", true))
				}
				userArr := []string{userID, userFirstName, userLastName}
				value = generatePasswordString(userID, userArr, buf2)
				if configDebug {
					buf2.WriteString(loggerGen(1, "Auto Generated Password for: "+userID+" - "+value, true))
				}
				if value == "" {
					buf2.WriteString(loggerGen(4, "Unable to generate password for: "+userID, true))
				}
			}
			value = base64.StdEncoding.EncodeToString([]byte(value))
			if configDebug {
				buf2.WriteString(loggerGen(1, "[PASSWORD] B64 encoded value: "+value, true))
			}
		}

		//-- if we have Value then set it
		if value != "" && value != "<nil>" && value != "&lt;nil&gt;" {
			espXmlmc.SetParam(field, value)

		}
	}

	//-- Check for Dry Run
	if !configDryRun {
		if configDebug {
			//-- DEBUG XML TO LOG FILE
			var XMLSTRING = espXmlmc.GetParam()
			buf2.WriteString(loggerGen(1, "User Create XML "+fmt.Sprintf("%v", XMLSTRING), true))
		}

		XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userCreate")
		var xmlRespon xmlmcResponse
		if xmlmcErr != nil {
			buf2.WriteString(loggerGen(4, "Error calling userCreate API: "+fmt.Sprintf("%v", xmlmcErr), true))
			errorCountInc()
			logger(0, buf2.String(), false)
			return
		}
		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			buf2.WriteString(loggerGen(4, "Error unmarshalling userCreate response: "+fmt.Sprintf("%v", err), true))
			errorCountInc()
			logger(0, buf2.String(), false)
			return
		}
		if xmlRespon.MethodResult != constOK {
			buf2.WriteString(loggerGen(4, "Error returned from userCreate: "+xmlRespon.State.ErrorRet, true))
			errorCountInc()
			logger(0, buf2.String(), false)
			return

		}
		buf2.WriteString(loggerGen(1, "User Create Success", true))

		//-- Only use Org lookup if enabled and not set to Update only
		if AzureImportConf.OrgLookup.Enabled && AzureImportConf.OrgLookup.Action != updateString && len(AzureImportConf.OrgLookup.OrgUnits) > 0 {
			userAddGroups(p, buf2, espXmlmc)
		}
		//-- Process Account Status
		if AzureImportConf.UserAccountStatus.Enabled && AzureImportConf.UserAccountStatus.Action != updateString {
			userSetStatus(userID, AzureImportConf.UserAccountStatus.Status, buf2, espXmlmc)
		}

		if AzureImportConf.UserRoleAction != updateString && len(AzureImportConf.Roles) > 0 {
			userAddRoles(userID, buf2, espXmlmc)
		}

		//-- Add Image
		if AzureImportConf.ImageLink.Enabled && AzureImportConf.ImageLink.Action != updateString && AzureImportConf.ImageLink.URI != "" {
			userAddImage(p, buf2, espXmlmc)
		}

		//-- Process Profile Details
		boolUpdateProfile := userUpdateProfile(p, buf2, espXmlmc)
		if !boolUpdateProfile {
			errorCountInc()
			logger(0, buf2.String(), false)
			return
		}

		createCountInc()
		logger(0, buf2.String(), false)
		return
	}

	//-- DEBUG XML TO LOG FILE
	var XMLSTRING = espXmlmc.GetParam()
	buf2.WriteString(loggerGen(1, "User Create XML "+fmt.Sprintf("%v", XMLSTRING), true))
	createSkippedCountInc()
	espXmlmc.ClearParam()
	logger(0, buf2.String(), false)
	return
}

func updateUser(u map[string]interface{}, espXmlmc *apiLib.XmlmcInstStruct) {
	buf2 := bytes.NewBufferString("")
	p := make(map[string]string)
	for key, value := range u {
		p[key] = fmt.Sprintf("%v", value)
	}
	userID := p[AzureImportConf.AzureConf.UserID]
	buf2.WriteString(loggerGen(1, "Update User: "+userID, true))
	for key := range userUpdateArray {
		field := userUpdateArray[key]
		value := AzureImportConf.UserMapping[field]

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
	if !configDryRun {
		if configDebug {
			//-- DEBUG XML TO LOG FILE
			var XMLSTRING = espXmlmc.GetParam()
			buf2.WriteString(loggerGen(1, "User Update XML "+fmt.Sprintf("%v", XMLSTRING), true))
		}
		XMLUpdate, xmlmcErr := espXmlmc.Invoke("admin", "userUpdate")
		var xmlRespon xmlmcResponse
		if xmlmcErr != nil {
			buf2.WriteString(loggerGen(4, "Error calling userUpdate API: "+fmt.Sprintf("%v", xmlmcErr), true))
			logger(0, buf2.String(), false)
			return
		}
		err := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			buf2.WriteString(loggerGen(4, "Error unmarshalling userUpdate API response: "+fmt.Sprintf("%v", err), true))
			logger(0, buf2.String(), false)
			return
		}

		if xmlRespon.MethodResult != constOK && xmlRespon.State.ErrorRet != noValuesToUpdate {
			buf2.WriteString(loggerGen(4, "Error returned from userUpdate API: "+xmlRespon.State.ErrorRet, true))
			errorCountInc()
			logger(0, buf2.String(), false)
			return

		}
		//-- Only use Org lookup if enabled and not set to create only
		if AzureImportConf.OrgLookup.Enabled && AzureImportConf.OrgLookup.Action != createString && len(AzureImportConf.OrgLookup.OrgUnits) > 0 {
			userAddGroups(p, buf2, espXmlmc)
		}
		//-- Process User Status
		if AzureImportConf.UserAccountStatus.Enabled && AzureImportConf.UserAccountStatus.Action != createString {
			userSetStatus(userID, AzureImportConf.UserAccountStatus.Status, buf2, espXmlmc)
		}

		//-- Add Roles
		if AzureImportConf.UserRoleAction != createString && len(AzureImportConf.Roles) > 0 {
			userAddRoles(userID, buf2, espXmlmc)
		}

		//-- Add Image
		if AzureImportConf.ImageLink.Enabled && AzureImportConf.ImageLink.Action != createString && AzureImportConf.ImageLink.URI != "" {
			userAddImage(p, buf2, espXmlmc)
		}

		//-- Process Profile Details
		boolUpdateProfile := userUpdateProfile(p, buf2, espXmlmc)
		if !boolUpdateProfile {
			errorCountInc()
			logger(0, buf2.String(), false)
			return
		}
		if xmlRespon.State.ErrorRet != noValuesToUpdate {
			buf2.WriteString(loggerGen(1, "User Update Success", true))
			updateCountInc()
		} else {
			updateSkippedCountInc()
		}
		logger(0, buf2.String(), false)
		return
	}
	//-- Inc Counter
	updateSkippedCountInc()
	//-- DEBUG XML TO LOG FILE
	var XMLSTRING = espXmlmc.GetParam()
	buf2.WriteString(loggerGen(1, "User Update XML "+fmt.Sprintf("%v", XMLSTRING), true))
	espXmlmc.ClearParam()
	logger(0, buf2.String(), false)
	return
}

func userAddGroups(p map[string]string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) bool {
	for _, orgUnit := range AzureImportConf.OrgLookup.OrgUnits {
		userAddGroup(p, buffer, orgUnit, espXmlmc)
	}
	return true
}
func userAddImage(p map[string]string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) {
	UserID := p[AzureImportConf.AzureConf.UserID]

	t := template.New("i" + UserID)
	t, _ = t.Parse(AzureImportConf.ImageLink.URI)
	buf := bytes.NewBufferString("")
	t.Execute(buf, p)
	value := buf.String()
	if value == "%!s(<nil>)" {
		value = ""
	}
	buffer.WriteString(loggerGen(2, "Image for user: "+value, true))
	if value == "" {
		return
	}

	strContentType := "image/jpeg"

	relLink := "session/" + UserID
	strDAVurl := AzureImportConf.DAVURL + relLink

	if strings.ToUpper(AzureImportConf.ImageLink.UploadType) != "URI" {
		// get binary to upload via WEBDAV and then set value to relative "session" URI
		client := http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Timeout: time.Duration(10 * time.Second),
		}

		var imageB []byte
		var Berr error
		switch strings.ToUpper(AzureImportConf.ImageLink.UploadType) {
		case "AZURE":

			strBearerToken, err := getBearerToken()
			if err != nil || strBearerToken == "" {
				buffer.WriteString(loggerGen(4, "[Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true))
				return
			}

			strURL := apiResource + "/" + AzureImportConf.AzureConf.APIVersion + "/users/" + strings.Replace(UserID, "@", "%40", -1) + "/thumbnailPhoto?"

			data := url.Values{}
			strData := data.Encode()
			strURL += strData
			req, _ := http.NewRequest("GET", strURL, nil)
			req.Header.Set("User-Agent", "Go-http-client/1.1")
			req.Header.Set("Authorization", "Bearer "+strBearerToken)
			duration := time.Second * time.Duration(30)
			imgclient := &http.Client{Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			}, Timeout: duration}
			resp, err := imgclient.Do(req)
			if err != nil {
				buffer.WriteString(loggerGen(4, " [IMAGE] Connection Error: "+fmt.Sprintf("%v", err), true))
				return
			}
			defer resp.Body.Close()

			//-- Check for HTTP Response
			if resp.StatusCode != 200 {
				if resp.StatusCode != 404 {
					errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
					buffer.WriteString(loggerGen(4, " [IMAGE] Error: "+errorString, true))
				} else {
					buffer.WriteString(loggerGen(4, " [IMAGE] Not Found", true))
				}
				//Drain the body so we can reuse the connection
				io.Copy(ioutil.Discard, resp.Body)
				return
			}
			buffer.WriteString(loggerGen(2, " [IMAGE] Connection Sucessful", true))

			imageB, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				buffer.WriteString(loggerGen(4, " [IMAGE] Cannot read the body of the response", true))
				return
			}
			strContentType = resp.Header.Get("Content-Type")

		case "URL":
			resp, err := http.Get(value)
			if err != nil {
				buffer.WriteString(loggerGen(4, "Unable to find "+value+" ["+fmt.Sprintf("%v", http.StatusInternalServerError)+"]", true))
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 201 || resp.StatusCode == 200 {
				imageB, _ = ioutil.ReadAll(resp.Body)

			} else {
				buffer.WriteString(loggerGen(4, "Unsuccesful download: "+fmt.Sprintf("%v", resp.StatusCode), true))
				return
			}

		default:
			imageB, Berr = hex.DecodeString(value[2:]) //stripping leading 0x
			if Berr != nil {
				buffer.WriteString(loggerGen(4, "Unsuccesful Decoding "+fmt.Sprintf("%v", Berr), true))
				return
			}

		}
		//WebDAV upload
		if len(imageB) > 0 {
			putbody := bytes.NewReader(imageB)
			req, _ := http.NewRequest("PUT", strDAVurl, putbody)
			req.Header.Set("Content-Type", strContentType)
			req.Header.Add("Authorization", "ESP-APIKEY "+AzureImportConf.APIKey)
			req.Header.Set("User-Agent", "Go-http-client/1.1")
			response, Perr := client.Do(req)
			if Perr != nil {
				buffer.WriteString(loggerGen(4, "PUT connection issue: "+fmt.Sprintf("%v", http.StatusInternalServerError), true))
				return
			}
			defer response.Body.Close()
			_, _ = io.Copy(ioutil.Discard, response.Body)
			if response.StatusCode == 201 || response.StatusCode == 200 {
				buffer.WriteString(loggerGen(1, "Uploaded", true))
				value = "/" + relLink
			} else {
				buffer.WriteString(loggerGen(4, "Unsuccesful Upload: "+fmt.Sprintf("%v", response.StatusCode), true))
				return
			}
		} else {
			buffer.WriteString(loggerGen(4, "No Image to upload", true))
			return
		}
	}

	espXmlmc.SetParam("objectRef", "urn:sys:user:"+UserID)
	espXmlmc.SetParam("sourceImage", value)

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("activity", "profileImageSet")
	var xmlRespon xmlmcprofileSetImageResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to associate Image to User Profile: "+fmt.Sprintf("%v", xmlmcErr), true))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Associate Image to User Profile: "+fmt.Sprintf("%v", err), true))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Associate Image to User Profile: "+xmlRespon.State.ErrorRet, true))
		} else {
			buffer.WriteString(loggerGen(1, "Image added to User: "+UserID, true))
		}
	}

	//Now go delete the file from dav
	reqDel, DelErr := http.NewRequest("DELETE", strDAVurl, nil)
	if DelErr != nil {
		buffer.WriteString(loggerGen(3, "User image updated but could not remove from session. Error: "+fmt.Sprintf("%v", DelErr), true))
		return
	}
	reqDel.Header.Add("Authorization", "ESP-APIKEY "+AzureImportConf.APIKey)
	reqDel.Header.Set("User-Agent", "Go-http-client/1.1")

	duration := time.Second * time.Duration(60)
	client := &http.Client{Timeout: duration}

	responseDel, DelErr := client.Do(reqDel)
	if DelErr != nil {
		buffer.WriteString(loggerGen(3, "User image updated but could not remove from session. Error: "+fmt.Sprintf("%v", DelErr), true))
		return
	}
	defer responseDel.Body.Close()
	_, _ = io.Copy(ioutil.Discard, responseDel.Body)
	if responseDel.StatusCode < 200 || responseDel.StatusCode > 299 {
		buffer.WriteString(loggerGen(3, "User image updated but could not remove from session. Status Code: "+strconv.Itoa(responseDel.StatusCode), true))
		return
	}
	buffer.WriteString(loggerGen(1, "User image removes from session successfully.", true))
}

func userAddGroup(p map[string]string, buffer *bytes.Buffer, orgUnit OrgUnitStruct, espXmlmc *apiLib.XmlmcInstStruct) bool {
	if orgUnit.Attribute == "" {
		buffer.WriteString(loggerGen(2, "Org Lookup is Enabled but Attribute is not Defined", true))
		return false
	}
	t := template.New("orgunit" + strconv.Itoa(orgUnit.Type))
	t, _ = t.Parse(orgUnit.Attribute)
	buf := bytes.NewBufferString("")
	t.Execute(buf, p)
	value := buf.String()
	if value == "%!s(<nil>)" {
		value = ""
	}
	buffer.WriteString(loggerGen(2, "Azure Attribute for Org Lookup: "+value, true))
	if value == "" {
		return true
	}

	orgAttributeName := processComplexField(value)
	orgIsInCache, orgID := groupInCache(strconv.Itoa(orgUnit.Type) + orgAttributeName)
	if orgIsInCache {
		buffer.WriteString(loggerGen(1, "Found Org in Cache "+orgID, true))
		userAddGroupAsoc(p, orgUnit, orgID, buffer, espXmlmc)
		return true
	}

	//-- We Get here if not in cache
	orgIsOnInstance, orgID := searchGroup(orgAttributeName, orgUnit, buffer, espXmlmc)
	if orgIsOnInstance {
		buffer.WriteString(loggerGen(1, "Org Lookup found Id "+orgID, true))
		userAddGroupAsoc(p, orgUnit, orgID, buffer, espXmlmc)
		return true
	}
	buffer.WriteString(loggerGen(1, "Unable to Find Organisation "+orgAttributeName, true))
	return false

}

func userAddGroupAsoc(p map[string]string, orgUnit OrgUnitStruct, orgID string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) {
	UserID := p[AzureImportConf.AzureConf.UserID]
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
		buffer.WriteString(loggerGen(4, "Unable to Associate User To Group: "+fmt.Sprintf("%v", xmlmcErr), true))
	}
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Associate User To Group: "+fmt.Sprintf("%v", err), true))
	} else {
		if xmlRespon.MethodResult != constOK {
			if xmlRespon.State.ErrorRet != "The specified user ["+UserID+"] already belongs to ["+orgID+"] group" {
				buffer.WriteString(loggerGen(4, "Unable to Associate User To Organisation: "+xmlRespon.State.ErrorRet, true))
			} else {
				buffer.WriteString(loggerGen(1, "User: "+UserID+" Already Added to Organisation: "+orgID, true))
			}

		} else {
			buffer.WriteString(loggerGen(1, "User: "+UserID+" Added to Organisation: "+orgID, true))
		}
	}

}

func userUpdateProfile(p map[string]string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) bool {
	UserID := p[AzureImportConf.AzureConf.UserID]
	buffer.WriteString(loggerGen(1, "Processing User Profile Data "+UserID, true))
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
	if !configDryRun {
		XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userProfileSet")
		var xmlRespon xmlmcResponse
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "Unable to Update User Profile: "+fmt.Sprintf("%v", xmlmcErr), true))
			return false
		}
		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Unable to Update User Profile: "+fmt.Sprintf("%v", err), true))

			return false
		}
		if xmlRespon.MethodResult != constOK {
			profileSkippedCountInc()
			if xmlRespon.State.ErrorRet == noValuesToUpdate {
				return true
			}
			err := errors.New(xmlRespon.State.ErrorRet)
			buffer.WriteString(loggerGen(4, "Unable to Update User Profile: "+fmt.Sprintf("%v", err), true))
			return false
		}
		profileCountInc()
		buffer.WriteString(loggerGen(1, "User Profile Update Success", true))
		return true

	}
	//-- DEBUG XML TO LOG FILE
	var XMLSTRING = espXmlmc.GetParam()
	buffer.WriteString(loggerGen(1, "User Profile Update XML "+fmt.Sprintf("%v", XMLSTRING), true))
	profileSkippedCountInc()
	espXmlmc.ClearParam()
	return true

}

func userSetStatus(userID string, status string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) bool {
	buffer.WriteString(loggerGen(1, "Set Status for User: "+fmt.Sprintf("%v", userID)+" Status:"+fmt.Sprintf("%v", status), true))

	espXmlmc.SetParam("userId", userID)
	espXmlmc.SetParam("accountStatus", status)

	XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userSetAccountStatus")

	var XMLSTRING = espXmlmc.GetParam()
	buffer.WriteString(loggerGen(1, "User Set Status XML "+fmt.Sprintf("%v", XMLSTRING), true))

	var xmlRespon xmlmcResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Set User Status: "+fmt.Sprintf("%v", xmlmcErr), true))
	}
	err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Set User Status "+fmt.Sprintf("%v", err), true))
		return false
	}
	if xmlRespon.MethodResult != constOK {
		if xmlRespon.State.ErrorRet != "Failed to update account status (target and the current status is the same)." {
			buffer.WriteString(loggerGen(4, "Unable to Set User Status ("+fmt.Sprintf("%v", status)+"): "+xmlRespon.State.ErrorRet, true))
			return false
		}
		buffer.WriteString(loggerGen(1, "User Status Already Set to: "+fmt.Sprintf("%v", status), true))
		return true
	}
	buffer.WriteString(loggerGen(1, "User Status Set Successfully", true))
	return true
}

func userAddRoles(userID string, buffer *bytes.Buffer, espXmlmc *apiLib.XmlmcInstStruct) bool {

	espXmlmc.SetParam("userId", userID)
	for _, role := range AzureImportConf.Roles {
		espXmlmc.SetParam("role", role)
		buffer.WriteString(loggerGen(1, "Add Role to User: "+role, true))
	}
	XMLCreate, xmlmcErr := espXmlmc.Invoke("admin", "userAddRole")
	var xmlRespon xmlmcResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Assign Role to User: "+fmt.Sprintf("%v", xmlmcErr), true))
	}
	err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to Assign Role to User: "+fmt.Sprintf("%v", err), true))
		return false
	}
	if xmlRespon.MethodResult != constOK {
		buffer.WriteString(loggerGen(4, "Unable to Assign Role to User: "+xmlRespon.State.ErrorRet, true))
		return false
	}
	buffer.WriteString(loggerGen(1, "Roles Added Successfully", true))
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

func getManagerFromLookup(manager string, buffer *bytes.Buffer) string {

	if manager == "" {
		buffer.WriteString(loggerGen(1, "No Manager to search", true))
		return ""
	}
	//-- Get Value of Attribute
	ManagerAttributeName := processComplexField(manager)
	buffer.WriteString(loggerGen(1, "Manager Lookup: "+ManagerAttributeName, true))

	//-- Dont Continue if we didn't get anything
	if ManagerAttributeName == "" {
		return ""
	}

	buffer.WriteString(loggerGen(1, "Looking Up Manager "+ManagerAttributeName, true))
	managerIsInCache, ManagerIDCache := managerInCache(ManagerAttributeName)

	//-- Check if we have Chached the site already
	if managerIsInCache {
		buffer.WriteString(loggerGen(1, "Found Manager in Cache "+ManagerIDCache, true))
		return ManagerIDCache
	}
	buffer.WriteString(loggerGen(1, "Manager Not In Cache Searching", true))
	ManagerIsOnInstance, ManagerIDInstance := searchManager(ManagerAttributeName, buffer)
	//-- If Returned set output
	if ManagerIsOnInstance {
		buffer.WriteString(loggerGen(1, "Manager Lookup found Id "+ManagerIDInstance, true))

		return ManagerIDInstance
	}

	return ""
}

//-- Search Manager on Instance
func searchManager(managerName string, buffer *bytes.Buffer) (bool, string) {
	boolReturn := false
	strReturn := ""
	//-- ESP Query for site
	espXmlmc := apiLib.NewXmlmcInstance(AzureImportConf.InstanceID)
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
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")
	XMLUserSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	var xmlRespon xmlmcUserListResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "Unable to Search for Manager: "+fmt.Sprintf("%v", xmlmcErr), true))
	}
	err := xml.Unmarshal([]byte(XMLUserSearch), &xmlRespon)
	if err != nil {
		stringError := err.Error()
		stringBody := string(XMLUserSearch)
		buffer.WriteString(loggerGen(4, "Unable to Search for Manager: "+fmt.Sprintf("%v", stringError+" RESPONSE BODY: "+stringBody), true))
	} else {
		if xmlRespon.MethodResult != constOK {
			buffer.WriteString(loggerGen(4, "Unable to Search for Manager: "+xmlRespon.State.ErrorRet, true))
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.UserName != "" {
				if strings.EqualFold(xmlRespon.Params.RowData.Row.UserID, managerName) {
					strReturn = xmlRespon.Params.RowData.Row.UserID
					boolReturn = true
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
		if strings.EqualFold(manager.UserName, managerName) {
			boolReturn = true
			stringReturn = manager.UserID
		}
	}
	mutexManagers.Unlock()
	return boolReturn, stringReturn
}
