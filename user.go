package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	apiLib "github.com/hornbill/goApiLib"
)

// Write DN and User ID to Cache
func writeUserToCache(DN string, ID string) {
	_, found := HornbillCache.DN[DN]
	if !found {
		HornbillCache.DN[DN] = ID
	}
}

// Get User ID From Cache By DN
func getUserFromDNCache(DN string) string {
	_, found := HornbillCache.DN[DN]
	if found {
		return HornbillCache.DN[DN]
	}
	return ""
}

func userCreate(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {
	buffer.WriteString(loggerGen(1, "User Create: "+user.Account.UserID))
	//-- Set Params based on already processed params
	hIF.SetParam("userId", user.Account.UserID)
	hIF.SetParam("name", user.Account.Name)
	//-- Password is base64 encoded already in process_data
	hIF.SetParam("password", user.Account.Password)
	hIF.SetParam("userType", user.Account.UserType)
	if user.Account.FirstName != "" {
		hIF.SetParam("firstName", user.Account.FirstName)
	}
	if user.Account.LastName != "" {
		hIF.SetParam("lastName", user.Account.LastName)
	}
	if user.Account.JobTitle != "" {
		hIF.SetParam("jobTitle", user.Account.JobTitle)
	}
	if user.Account.Site != "" {
		hIF.SetParam("site", user.Account.Site)
	}
	if user.Account.Phone != "" {
		hIF.SetParam("phone", user.Account.Phone)
	}
	if user.Account.Email != "" {
		hIF.SetParam("email", user.Account.Email)
	}
	if user.Account.Mobile != "" {
		hIF.SetParam("mobile", user.Account.Mobile)
	}
	//hIF.SetParam("availabilityStatus", 1)
	if user.Account.AbsenceMessage != "" {
		hIF.SetParam("absenceMessage", user.Account.AbsenceMessage)
	}
	if user.Account.TimeZone != "" {
		hIF.SetParam("timeZone", user.Account.TimeZone)
	}
	if user.Account.Language != "" {
		hIF.SetParam("language", user.Account.Language)
	}
	if user.Account.DateTimeFormat != "" {
		hIF.SetParam("dateTimeFormat", user.Account.DateTimeFormat)
	}
	if user.Account.DateFormat != "" {
		hIF.SetParam("dateFormat", user.Account.DateFormat)
	}
	if user.Account.TimeFormat != "" {
		hIF.SetParam("timeFormat", user.Account.TimeFormat)
	}
	if user.Account.CurrencySymbol != "" {
		hIF.SetParam("currencySymbol", user.Account.CurrencySymbol)
	}
	if user.Account.CountryCode != "" {
		hIF.SetParam("countryCode", user.Account.CountryCode)
	}
	//hIF.SetParam("notifyEmail", "")
	//hIF.SetParam("notifyTextMessage", "")

	//-- Dry Run
	if Flags.configDryRun {
		var XMLSTRING = hIF.GetParam()

		buffer.WriteString(loggerGen(1, "User Create XML "+XMLSTRING))
		hIF.ClearParam()
		return true, nil
	}
	var XMLSTRING = hIF.GetParam()
	fmt.Println(XMLSTRING)

	RespBody, xmlmcErr := hIF.Invoke("admin", "userCreate")
	var JSONResp xmlmcResponse
	if xmlmcErr != nil {
		return false, xmlmcErr
	}
	fmt.Println(RespBody)
	return true, nil
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		return false, err
	}
	if JSONResp.State.Error != "" {
		return false, errors.New(JSONResp.State.Error)
	}
	buffer.WriteString(loggerGen(1, "User Create Success: "+user.Account.UserID))
	return true, nil
}

func userUpdate(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {

	buffer.WriteString(loggerGen(1, "User Update: "+user.Account.UserID))
	//-- Set Params based on already processed params
	hIF.SetParam("userId", user.Account.UserID)
	hIF.SetParam("userType", user.Account.UserType)
	hIF.SetParam("name", user.Account.Name)
	//hIF.SetParam("password", user.Password)
	if user.Account.FirstName != "" {
		hIF.SetParam("firstName", user.Account.FirstName)
	}
	if user.Account.LastName != "" {
		hIF.SetParam("lastName", user.Account.LastName)
	}
	if user.Account.JobTitle != "" {
		hIF.SetParam("jobTitle", user.Account.JobTitle)
	}
	if user.Account.Site != "" {
		hIF.SetParam("site", user.Account.Site)
	}
	if user.Account.Phone != "" {
		hIF.SetParam("phone", user.Account.Phone)
	}
	if user.Account.Email != "" {
		hIF.SetParam("email", user.Account.Email)
	}
	if user.Account.Mobile != "" {
		hIF.SetParam("mobile", user.Account.Mobile)
	}
	//hIF.SetParam("availabilityStatus", 1)
	if user.Account.AbsenceMessage != "" {
		hIF.SetParam("absenceMessage", user.Account.AbsenceMessage)
	}
	if user.Account.TimeZone != "" {
		hIF.SetParam("timeZone", user.Account.TimeZone)
	}
	if user.Account.Language != "" {
		hIF.SetParam("language", user.Account.Language)
	}
	if user.Account.DateTimeFormat != "" {
		hIF.SetParam("dateTimeFormat", user.Account.DateTimeFormat)
	}
	if user.Account.DateFormat != "" {
		hIF.SetParam("dateFormat", user.Account.DateFormat)
	}
	if user.Account.TimeFormat != "" {
		hIF.SetParam("timeFormat", user.Account.TimeFormat)
	}
	if user.Account.CurrencySymbol != "" {
		hIF.SetParam("currencySymbol", user.Account.CurrencySymbol)
	}
	if user.Account.CountryCode != "" {
		hIF.SetParam("countryCode", user.Account.CountryCode)
	}
	//hIF.SetParam("notifyEmail", "")
	//hIF.SetParam("notifyTextMessage", "")
	var XMLSTRING = hIF.GetParam()
	//-- Dry Run
	if Flags.configDryRun {
		buffer.WriteString(loggerGen(1, "User Update XML "+XMLSTRING))
		hIF.ClearParam()
		return true, nil
	}

	RespBody, xmlmcErr := hIF.Invoke("admin", "userUpdate")
	var JSONResp xmlmcResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(1, "User Update Profile XML "+XMLSTRING))
		return false, xmlmcErr
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		buffer.WriteString(loggerGen(1, "User Update Profile XML "+XMLSTRING))
		return false, err
	}
	if JSONResp.State.Error != "" {
		buffer.WriteString(loggerGen(1, "User Update Profile XML "+XMLSTRING))
		return false, errors.New(JSONResp.State.Error)
	}
	buffer.WriteString(loggerGen(1, "User Update Success: "+user.Account.UserID))
	return true, nil
}

func userProfileUpdate(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {
	buffer.WriteString(loggerGen(1, "User Update Profile: "+user.Account.UserID))

	hIF.OpenElement("profileData")

	//-- Set Params based on already processed params
	hIF.SetParam("userId", user.Account.UserID)
	if user.Profile.MiddleName != "" {
		hIF.SetParam("middleName", user.Profile.MiddleName)
	}
	if user.Profile.JobDescription != "" {
		hIF.SetParam("jobDescription", user.Profile.JobDescription)
	}
	if user.Profile.Manager != "" {
		hIF.SetParam("manager", user.Profile.Manager)
	}
	if user.Profile.WorkPhone != "" {
		hIF.SetParam("workPhone", user.Profile.WorkPhone)
	}
	if user.Profile.Qualifications != "" {
		hIF.SetParam("qualifications", user.Profile.Qualifications)
	}
	if user.Profile.Interests != "" {
		hIF.SetParam("interests", user.Profile.Interests)
	}
	if user.Profile.Expertise != "" {
		hIF.SetParam("expertise", user.Profile.Expertise)
	}
	if user.Profile.Gender != "" {
		hIF.SetParam("gender", user.Profile.Gender)
	}
	if user.Profile.Dob != "" {
		hIF.SetParam("dob", user.Profile.Dob)
	}
	if user.Profile.Nationality != "" {
		hIF.SetParam("nationality", user.Profile.Nationality)
	}
	if user.Profile.Religion != "" {
		hIF.SetParam("religion", user.Profile.Religion)
	}
	if user.Profile.HomeTelephone != "" {
		hIF.SetParam("homeTelephone", user.Profile.HomeTelephone)
	}
	if user.Profile.SocialNetworkA != "" {
		hIF.SetParam("socialNetworkA", user.Profile.SocialNetworkA)
	}
	if user.Profile.SocialNetworkB != "" {
		hIF.SetParam("socialNetworkB", user.Profile.SocialNetworkB)
	}
	if user.Profile.SocialNetworkC != "" {
		hIF.SetParam("socialNetworkC", user.Profile.SocialNetworkC)
	}
	if user.Profile.SocialNetworkD != "" {
		hIF.SetParam("socialNetworkD", user.Profile.SocialNetworkD)
	}
	if user.Profile.SocialNetworkE != "" {
		hIF.SetParam("socialNetworkE", user.Profile.SocialNetworkE)
	}
	if user.Profile.SocialNetworkF != "" {
		hIF.SetParam("socialNetworkF", user.Profile.SocialNetworkF)
	}
	if user.Profile.SocialNetworkG != "" {
		hIF.SetParam("socialNetworkG", user.Profile.SocialNetworkG)
	}
	if user.Profile.SocialNetworkH != "" {
		hIF.SetParam("socialNetworkH", user.Profile.SocialNetworkH)
	}
	if user.Profile.PersonalInterests != "" {
		hIF.SetParam("personalInterests", user.Profile.PersonalInterests)
	}
	if user.Profile.HomeAddress != "" {
		hIF.SetParam("homeAddress", user.Profile.HomeAddress)
	}
	if user.Profile.PersonalBlog != "" {
		hIF.SetParam("personalBlog", user.Profile.PersonalBlog)
	}
	if user.Profile.Attrib1 != "" {
		hIF.SetParam("attrib1", user.Profile.Attrib1)
	}
	if user.Profile.Attrib2 != "" {
		hIF.SetParam("attrib2", user.Profile.Attrib2)
	}
	if user.Profile.Attrib3 != "" {
		hIF.SetParam("attrib3", user.Profile.Attrib3)
	}
	if user.Profile.Attrib4 != "" {
		hIF.SetParam("attrib4", user.Profile.Attrib4)
	}
	if user.Profile.Attrib5 != "" {
		hIF.SetParam("attrib5", user.Profile.Attrib5)
	}
	if user.Profile.Attrib6 != "" {
		hIF.SetParam("attrib6", user.Profile.Attrib6)
	}
	if user.Profile.Attrib7 != "" {
		hIF.SetParam("attrib7", user.Profile.Attrib7)
	}
	if user.Profile.Attrib8 != "" {
		hIF.SetParam("attrib8", user.Profile.Attrib8)
	}

	hIF.CloseElement("profileData")
	var XMLSTRING = hIF.GetParam()
	//-- Dry Run
	if Flags.configDryRun {
		buffer.WriteString(loggerGen(1, "User Update Profile XML "+XMLSTRING))
		hIF.ClearParam()
		return true, nil
	}

	RespBody, xmlmcErr := hIF.Invoke("admin", "userProfileSet")
	var JSONResp xmlmcResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(1, "User Update Profile XML "+XMLSTRING))
		return false, xmlmcErr
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		buffer.WriteString(loggerGen(1, "User Update Profile XML "+XMLSTRING))
		return false, err
	}
	if JSONResp.State.Error != "" {
		buffer.WriteString(loggerGen(1, "User Update Profile XML "+XMLSTRING))
		return false, errors.New(JSONResp.State.Error)
	}
	buffer.WriteString(loggerGen(1, "User Update Profile Success: "+user.Account.UserID))
	return true, nil
}

func userRolesUpdate(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {

	hIF.SetParam("userId", user.Account.UserID)
	for roleIndex := range user.Roles {
		role := user.Roles[roleIndex]
		buffer.WriteString(loggerGen(1, "User Add Role User: "+user.Account.UserID+" Role: "+role))
		hIF.SetParam("role", role)
	}
	var XMLSTRING = hIF.GetParam()
	if Flags.configDryRun {
		buffer.WriteString(loggerGen(1, "User Add Role XML "+XMLSTRING))
		hIF.ClearParam()
		return true, nil
	}

	RespBody, xmlmcErr := hIF.Invoke("admin", "userAddRole")
	var JSONResp xmlmcResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(1, "User Add Role XML "+XMLSTRING))
		return false, xmlmcErr
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		buffer.WriteString(loggerGen(1, "User Add Role XML "+XMLSTRING))
		return false, err
	}
	if JSONResp.State.Error != "" {
		buffer.WriteString(loggerGen(1, "User Add Role XML "+XMLSTRING))
		return false, errors.New(JSONResp.State.Error)
	}
	buffer.WriteString(loggerGen(1, "Role added to User: "+user.Account.UserID))
	return true, nil
}

func userStatusUpdate(hIF *apiLib.XmlmcInstStruct, user *userWorkingDataStruct, buffer *bytes.Buffer) (bool, error) {

	hIF.SetParam("userId", user.Account.UserID)
	hIF.SetParam("accountStatus", AzureImportConf.User.Status.Value)

	var XMLSTRING = hIF.GetParam()
	if Flags.configDryRun {
		buffer.WriteString(loggerGen(1, "User Set Status XML "+XMLSTRING))
		hIF.ClearParam()
		return true, nil
	}

	RespBody, xmlmcErr := hIF.Invoke("admin", "userSetAccountStatus")
	var JSONResp xmlmcResponse
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(1, "User Set Status XML "+XMLSTRING))
		return false, xmlmcErr
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		buffer.WriteString(loggerGen(1, "User Set Status XML "+XMLSTRING))
		return false, err
	}
	if JSONResp.State.Error != "" {
		buffer.WriteString(loggerGen(1, "User Set Status XML "+XMLSTRING))
		return false, errors.New(JSONResp.State.Error)
	}
	buffer.WriteString(loggerGen(1, "User Status Updated: "+user.Account.UserID))
	return true, nil
}
