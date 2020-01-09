package main

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

//-- Store DB Users in Map
func processDBUsers() {
	logger(1, "Processing DB User Data", true)

	//-- User Working Data
	HornbillCache.UsersWorking = make(map[string]*userWorkingDataStruct)
	HornbillCache.Managers = make(map[string]string)
	HornbillCache.DN = make(map[string]string)
	HornbillCache.Images = make(map[string]imageStruct)
	//-- Loop DB Users
	for user := range localDBUsers {
		// Process Pre Import Actions
		var userID = processImportActions(&localDBUsers[user])
		// Process Params and return userId
		processUserParams(&localDBUsers[user], userID)
		if userID != "" {
			var userDN = processComplexField(&localDBUsers[user], AzureImportConf.User.UserDN)
			//-- Write to Cache
			writeUserToCache(userDN, userID)
		}
	}

	logger(1, "DB Users Processed: "+fmt.Sprintf("%d", len(localDBUsers))+"\n", true)
}

/*
func processDBUsers() {
	logger(1, "Processing DB User Data", true)

	//-- User Working Data
	HornbillCache.UsersWorking = make(map[string]*userWorkingDataStruct)
	HornbillCache.Managers = make(map[string]string)
	HornbillCache.DN = make(map[string]string)
	HornbillCache.Images = make(map[string]imageStruct)
	//-- Loop DB Users
	for user := range ldapUsers {
		// Process Pre Import Actions
		var userID = processImportActions(ldapUsers[user])
		// Process Params and return userId
		processUserParams(ldapUsers[user], userID)
		if userID != "" {
			var userDN = processComplexField(ldapUsers[user], AzureImportConf.User.UserDN)
			//-- Write to Cache
			writeUserToCache(userDN, userID)
		}
	}

	logger(1, "DB Users Processed: "+fmt.Sprintf("%d", len(ldapUsers))+"\n", true)
}
*/

func processData() {
	logger(1, "Processing User Data", true)

	for user := range HornbillCache.UsersWorking {

		currentUser := HornbillCache.UsersWorking[user]
		//-- Current UserID
		userID := strings.ToLower(currentUser.Account.UserID)

		//-- Extra Debugging
		logger(1, "DB User ID: '"+userID+"'\n", false)

		hornbillUserData := HornbillCache.Users[userID]

		if userID == "" {
			CounterInc(7)
			logger(4, "DB Record Has no User ID: '"+fmt.Sprintf("%+v", currentUser.DB)+"'\n", false)
			continue
		}
		//-- Check Map no need to loop
		if strings.ToLower(hornbillUserData.HUserID) == userID {

			currentUser.Jobs.update = checkUserNeedsUpdate(currentUser, hornbillUserData)

			currentUser.Jobs.updateProfile = checkUserNeedsProfileUpdate(currentUser, hornbillUserData)

			currentUser.Jobs.updateType = checkUserNeedsTypeUpdate(currentUser, hornbillUserData)

			currentUser.Jobs.updateSite = checkUserNeedsSiteUpdate(currentUser, hornbillUserData)

			currentUser.Jobs.updateImage = checkUserNeedsImageUpdate(currentUser, hornbillUserData)

			currentUser.Jobs.updateHomeOrg = checkUserNeedsHomeOrgUpdate(currentUser, hornbillUserData)

			checkUserNeedsOrgUpdate(currentUser, hornbillUserData)

			checkUserNeedsOrgRemoving(currentUser, hornbillUserData)

			checkUserNeedsRoleUpdate(currentUser, hornbillUserData)

			currentUser.Jobs.updateStatus = checkUserNeedsStatusUpdate(currentUser, hornbillUserData)
		} else {
			//-- Check for Password
			setUserPasswordValueForCreate(currentUser)
			//-- Set Site ID Based on Config
			setUserSiteValueForCreate(currentUser, hornbillUserData)
			setUserRolesalueForCreate(currentUser, hornbillUserData)
			currentUser.Jobs.updateImage = checkUserNeedsImageCreate(currentUser, hornbillUserData)
			checkUserNeedsOrgCreate(currentUser, hornbillUserData)
			currentUser.Jobs.updateStatus = checkUserNeedsStatusCreate(currentUser, hornbillUserData)
			currentUser.Jobs.create = true
		}

		loggerOutput := []string{
			"User: " + userID,
			"Create: " + strconv.FormatBool(currentUser.Jobs.create),
			"Update: " + strconv.FormatBool(currentUser.Jobs.update),
			"Update Type: " + strconv.FormatBool(currentUser.Jobs.updateType),
			"Update Profile: " + strconv.FormatBool(currentUser.Jobs.updateProfile),
			"Update Site: " + strconv.FormatBool(currentUser.Jobs.updateSite),
			"Update Status: " + strconv.FormatBool(currentUser.Jobs.updateStatus),
			"Update Home Organisation: " + strconv.FormatBool(currentUser.Jobs.updateHomeOrg),
			"Roles Count: " + fmt.Sprintf("%d", len(currentUser.Roles)),
			"Update Image: " + strconv.FormatBool(currentUser.Jobs.updateImage),
			"Groups: " + fmt.Sprintf("%d", len(currentUser.Groups))}

		strings.Join(loggerOutput[:], "\n\t")
		logger(1, strings.Join(loggerOutput[:], "\n\t")+"\n", false)
	}
	logger(1, "User Data Processed: "+fmt.Sprintf("%d", len(HornbillCache.UsersWorking))+"", true)
}

func checkUserNeedsStatusCreate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {

	if AzureImportConf.User.Role.Action == "Both" || AzureImportConf.User.Role.Action == "Create" {
		//-- By default they are created active so if we need to change the status it should be done if not active
		if AzureImportConf.User.Status.Value != "active" {
			return true
		}
	}

	return false
}
func checkUserNeedsStatusUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {

	if AzureImportConf.User.Status.Action == "Both" || AzureImportConf.User.Status.Action == "Update" {
		//-- Check current status != config status
		if HornbillUserStatusMap[currentData.HAccountStatus] != AzureImportConf.User.Status.Value {
			return true
		}
	}
	return false
}
func setUserPasswordValueForCreate(importData *userWorkingDataStruct) {
	if importData.Account.Password == "" {
		//-- Generate Password
		importData.Account.Password = generatePasswordString(importData)
		logger(1, "Auto Generated Password for: "+importData.Account.UserID+" - "+importData.Account.Password, false)
	}
	//-- Base64 Encode
	importData.Account.Password = base64.StdEncoding.EncodeToString([]byte(importData.Account.Password))
}
func checkUserNeedsOrgRemoving(importData *userWorkingDataStruct, currentData userAccountStruct) {
	//-- Only if we have some config for groups
	if len(AzureImportConf.User.Org) > 0 {

		//-- List of Existing Groups
		var userExistingGroups = HornbillCache.UserGroups[strings.ToLower(importData.Account.UserID)]

		for index := range userExistingGroups {
			ExistingGroupID := userExistingGroups[index]
			ExistingGroup := HornbillCache.GroupsID[strings.ToLower(ExistingGroupID)]
			boolGroupNeedsRemoving := false

			//-- Loop Config Orgs and Check each one
			for orgIndex := range AzureImportConf.User.Org {

				//-- Get Group from Index
				importOrg := AzureImportConf.User.Org[orgIndex]

				//-- Only if Actions is correct
				if importOrg.Action == "Both" || importOrg.Action == "Update" {
					//-- Evaluate the Id
					var GroupID = getOrgFromLookup(importData, importOrg.Value, importOrg.Options.Type)
					//-- If already a member of import group then ignore
					if GroupID == ExistingGroup.ID {
						//-- exit for loop
						continue
					}

					//-- If group we are a memember of matches the Type of a group we have set up on the import and its set to one Assignment
					if importOrg.Options.Type == ExistingGroup.Type && importOrg.Options.OnlyOneGroupAssignment {
						boolGroupNeedsRemoving = true
					}
				}
			}
			//-- If group is not part of import and its set to remove
			if boolGroupNeedsRemoving {
				importData.GroupsToRemove = append(importData.GroupsToRemove, ExistingGroupID)
			}
		}
	}
}

func checkUserNeedsOrgUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) {
	if len(AzureImportConf.User.Org) > 0 {
		for orgIndex := range AzureImportConf.User.Org {
			orgAction := AzureImportConf.User.Org[orgIndex]
			if orgAction.Action == "Both" || orgAction.Action == "Update" {
				var GroupID = getOrgFromLookup(importData, orgAction.Value, orgAction.Options.Type)
				var userExistingGroups = HornbillCache.UserGroups[strings.ToLower(importData.Account.UserID)]
				//-- Is User Already a Memeber of the Group
				boolUserInGroup := false
				for index := range userExistingGroups {
					if strings.EqualFold(GroupID, userExistingGroups[index]) {
						boolUserInGroup = true
					}
				}
				if !boolUserInGroup && GroupID != "" {
					//-- Check User is a member of
					if orgAction.MemberOf != "" {
						if !isUserAMember(importData.DB, orgAction.MemberOf) {
							continue
						}
					}
					var group userGroupStruct
					group.ID = GroupID
					group.Name = orgAction.Value
					group.Type = orgAction.Options.Type
					group.Membership = orgAction.Options.Membership
					group.TasksView = orgAction.Options.TasksView
					group.TasksAction = orgAction.Options.TasksAction
					group.OnlyOneGroupAssignment = orgAction.Options.OnlyOneGroupAssignment

					importData.Groups = append(importData.Groups, group)
				}
			}
		}
	}
}

func checkUserNeedsHomeOrgUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if len(AzureImportConf.User.Org) > 0 {
		for orgIndex := range AzureImportConf.User.Org {
			orgAction := AzureImportConf.User.Org[orgIndex]
			if !orgAction.Options.SetAsHomeOrganisation {
				continue
			}
			if orgAction.Action == "Create" || orgAction.Action == "Both" || orgAction.Action == "Update" {
				var GroupID = getOrgFromLookup(importData, orgAction.Value, orgAction.Options.Type)

				if GroupID == "" || strings.EqualFold(currentData.HHomeOrg, GroupID) {
					return false
				}
				importData.Account.HomeOrg = GroupID
				logger(1, "Home Organisation: "+GroupID+" - "+currentData.HHomeOrg, true)
				return true
			}
		}
	}
	return false
}

func checkUserNeedsOrgCreate(importData *userWorkingDataStruct, currentData userAccountStruct) {
	if len(AzureImportConf.User.Org) > 0 {
		for orgIndex := range AzureImportConf.User.Org {
			orgAction := AzureImportConf.User.Org[orgIndex]
			if orgAction.Action == "Both" || orgAction.Action == "Create" {

				var GroupID = getOrgFromLookup(importData, orgAction.Value, orgAction.Options.Type)

				if GroupID != "" && orgAction.MemberOf != "" {
					if !isUserAMember(importData.DB, orgAction.MemberOf) {
						continue
					}
				}
				var group userGroupStruct
				group.ID = GroupID
				group.Name = orgAction.Value
				group.Type = orgAction.Options.Type
				group.Membership = orgAction.Options.Membership
				group.TasksView = orgAction.Options.TasksView
				group.TasksAction = orgAction.Options.TasksAction
				group.OnlyOneGroupAssignment = orgAction.Options.OnlyOneGroupAssignment

				if GroupID != "" {
					importData.Groups = append(importData.Groups, group)
				}
			}
		}
	}
}
func setUserRolesalueForCreate(importData *userWorkingDataStruct, currentData userAccountStruct) {

	if AzureImportConf.User.Role.Action == "Both" || AzureImportConf.User.Role.Action == "Create" {
		importData.Roles = AzureImportConf.User.Role.Roles
	}
}
func checkUserNeedsRoleUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) {

	if AzureImportConf.User.Role.Action == "Both" || AzureImportConf.User.Role.Action == "Update" {
		for index := range AzureImportConf.User.Role.Roles {
			roleName := AzureImportConf.User.Role.Roles[index]
			foundRole := false
			var userRoles = HornbillCache.UserRoles[strings.ToLower(importData.Account.UserID)]
			for index2 := range userRoles {
				if strings.EqualFold(roleName, userRoles[index2]) {
					foundRole = true
				}
			}
			if !foundRole {
				importData.Roles = append(importData.Roles, roleName)
			}
		}
	}
}
func checkUserNeedsImageCreate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	//-- Is Type Enables for Update or both
	if AzureImportConf.User.Image.Action == "Both" || AzureImportConf.User.Image.Action == "Create" {

		//-- Check for Empty URI
		if AzureImportConf.User.Image.URI == "" {
			return false
		}
		image := getImage(importData)
		// check for changes
		if image.imageCheckSum != currentData.HIconChecksum {
			return true
		}
	}
	return false
}
func checkUserNeedsImageUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	//-- Is Type Enables for Update or both
	if AzureImportConf.User.Image.Action == "Both" || AzureImportConf.User.Image.Action == "Update" {

		//-- Check for Empty URI
		if AzureImportConf.User.Image.URI == "" {
			return false
		}
		image := getImage(importData)
		// check for changes
		if image.imageCheckSum != currentData.HIconChecksum {
			return true
		}
	}
	return false
}
func checkUserNeedsTypeUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	//-- Is Type Enables for Update or both
	if AzureImportConf.User.Type.Action == "Both" || AzureImportConf.User.Type.Action == "Update" {
		// -- 1 = user
		// -- 3 = basic
		switch importData.Account.UserType {
		case "user":
			if currentData.HClass != "1" {
				return true
			}
		case "basic":
			if currentData.HClass != "3" {
				return true
			}
		default:
			return false
		}
	} else {
		if currentData.HClass == "1" {
			importData.Account.UserType = "user"
		} else {
			importData.Account.UserType = "basic"
		}
	}
	return false
}
func setUserSiteValueForCreate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	//-- Is Site Enables for Update or both
	if AzureImportConf.User.Site.Action == "Both" || AzureImportConf.User.Site.Action == "Create" {
		importData.Account.Site = getSiteFromLookup(importData)
	}
	if importData.Account.Site != "" && importData.Account.Site != currentData.HSite {
		return true
	}
	return false
}
func checkUserNeedsSiteUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	//-- Is Site Enables for Update or both
	if AzureImportConf.User.Site.Action == "Both" || AzureImportConf.User.Site.Action == "Update" {
		importData.Account.Site = getSiteFromLookup(importData)
	} else {
		//-- Else Default to current value
		importData.Account.Site = currentData.HSite
	}

	if importData.Account.Site != "" && importData.Account.Site != currentData.HSite {
		return true
	}
	return false
}
func checkUserNeedsUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {
	if importData.Account.LoginID != "" && importData.Account.LoginID != currentData.HLoginID {
		logger(1, "LoginID: "+importData.Account.LoginID+" - "+currentData.HLoginID, true)
		return true
	}
	if importData.Account.EmployeeID != "" && importData.Account.EmployeeID != currentData.HEmployeeID {
		logger(1, "EmployeeID: "+importData.Account.EmployeeID+" - "+currentData.HEmployeeID, true)
		return true
	}
	if importData.Account.Name != "" && importData.Account.Name != currentData.HName {
		logger(1, "Name: "+importData.Account.Name+" - "+currentData.HName, true)
		return true
	}
	if importData.Account.FirstName != "" && importData.Account.FirstName != currentData.HFirstName {
		logger(1, "FirstName: "+importData.Account.FirstName+" - "+currentData.HFirstName, true)
		return true
	}
	if importData.Account.LastName != "" && importData.Account.LastName != currentData.HLastName {
		logger(1, "LastName: "+importData.Account.LastName+" - "+currentData.HLastName, true)
		return true
	}
	if importData.Account.JobTitle != "" && importData.Account.JobTitle != currentData.HJobTitle {
		logger(1, "JobTitle: "+importData.Account.JobTitle+" - "+currentData.HJobTitle, true)
		return true
	}
	if importData.Account.Phone != "" && importData.Account.Phone != currentData.HPhone {
		logger(1, "Phone: "+importData.Account.Phone+" - "+currentData.HPhone, true)
		return true
	}
	if importData.Account.Email != "" && importData.Account.Email != currentData.HEmail {
		logger(1, "Email: "+importData.Account.Email+" - "+currentData.HEmail, true)
		return true
	}
	if importData.Account.Mobile != "" && importData.Account.Mobile != currentData.HMobile {
		logger(1, "Mobile: "+importData.Account.Mobile+" - "+currentData.HMobile, true)
		return true
	}
	if importData.Account.AbsenceMessage != "" && importData.Account.AbsenceMessage != currentData.HAvailStatusMsg {
		logger(1, "AbsenceMessage: "+importData.Account.AbsenceMessage+" - "+currentData.HAvailStatusMsg, true)
		return true
	}
	//-- If TimeZone mapping is empty then ignore as it defaults to a value
	if importData.Account.TimeZone != "" && importData.Account.TimeZone != currentData.HTimezone {
		logger(1, "TimeZone: "+importData.Account.TimeZone+" - "+currentData.HTimezone, true)
		return true
	}
	//-- If Language mapping is empty then ignore as it defaults to a value
	if importData.Account.Language != "" && importData.Account.Language != currentData.HLanguage {
		logger(1, "Language: "+importData.Account.Language+" - "+currentData.HLanguage, true)
		return true
	}
	//-- If DateTimeFormat mapping is empty then ignore as it defaults to a value
	if importData.Account.DateTimeFormat != "" && importData.Account.DateTimeFormat != currentData.HDateTimeFormat {
		logger(1, "DateTimeFormat: "+importData.Account.DateTimeFormat+" - "+currentData.HDateTimeFormat, true)
		return true
	}
	//-- If DateFormat mapping is empty then ignore as it defaults to a value
	if importData.Account.DateFormat != "" && importData.Account.DateFormat != currentData.HDateFormat {
		logger(1, "DateFormat: "+importData.Account.DateFormat+" - "+currentData.HDateFormat, true)
		return true
	}
	//-- If TimeFormat mapping is empty then ignore as it defaults to a value
	if importData.Account.TimeFormat != "" && importData.Account.TimeFormat != currentData.HTimeFormat {
		logger(1, "TimeFormat: "+importData.Account.TimeFormat+" - "+currentData.HTimeFormat, true)
		return true
	}
	//-- If CurrencySymbol mapping is empty then ignore as it defaults to a value
	if importData.Account.CurrencySymbol != "" && importData.Account.CurrencySymbol != currentData.HCurrencySymbol {
		logger(1, "CurrencySymbol: "+importData.Account.CurrencySymbol+" - "+currentData.HCurrencySymbol, true)
		return true
	}
	//-- If CountryCode mapping is empty then ignore as it defaults to a value
	if importData.Account.CountryCode != "" && importData.Account.CountryCode != currentData.HCountry {
		logger(1, "CountryCode: "+importData.Account.CountryCode+" - "+currentData.HCountry, true)
		return true
	}

	return false
}
func checkUserNeedsProfileUpdate(importData *userWorkingDataStruct, currentData userAccountStruct) bool {

	if importData.Profile.MiddleName != "" && importData.Profile.MiddleName != currentData.HMiddleName {
		logger(1, "MiddleName: "+importData.Profile.MiddleName+" - "+currentData.HMiddleName, true)
		return true
	}

	if importData.Profile.JobDescription != "" && importData.Profile.JobDescription != currentData.HSummary {
		logger(1, "JobDescription: "+importData.Profile.JobDescription+" - "+currentData.HSummary, true)
		return true
	}
	if AzureImportConf.User.Manager.Action == "Both" || AzureImportConf.User.Manager.Action == "Update" {
		importData.Profile.Manager = getManager(importData, currentData)
	} else {
		//-- Use Current Value
		importData.Profile.Manager = currentData.HManager
	}
	if importData.Profile.Manager != "" && importData.Profile.Manager != currentData.HManager {
		logger(1, "Manager: "+importData.Profile.Manager+" - "+currentData.HManager, true)
		return true
	}
	if importData.Profile.WorkPhone != "" && importData.Profile.WorkPhone != currentData.HPhone {
		logger(1, "WorkPhone: "+importData.Profile.WorkPhone+" - "+currentData.HPhone, true)
		return true
	}
	if importData.Profile.Qualifications != "" && importData.Profile.Qualifications != currentData.HQualifications {
		logger(1, "Qualifications: "+importData.Profile.Qualifications+" - "+currentData.HQualifications, true)
		return true
	}
	if importData.Profile.Interests != "" && importData.Profile.Interests != currentData.HInterests {
		logger(1, "Interests: "+importData.Profile.Interests+" - "+currentData.HInterests, true)
		return true
	}
	if importData.Profile.Expertise != "" && importData.Profile.Expertise != currentData.HSkills {
		logger(1, "Expertise: "+importData.Profile.Expertise+" - "+currentData.HSkills, true)
		return true
	}
	if importData.Profile.Gender != "" && importData.Profile.Gender != currentData.HGender {
		logger(1, "Gender: "+importData.Profile.Gender+" - "+currentData.HGender, true)
		return true
	}
	if importData.Profile.Dob != "" && importData.Profile.Dob != currentData.HDob {
		logger(1, "Dob: "+importData.Profile.Dob+" - "+currentData.HDob, true)
		return true
	}
	if importData.Profile.Nationality != "" && importData.Profile.Nationality != currentData.HNationality {
		logger(1, "Nationality: "+importData.Profile.Nationality+" - "+currentData.HNationality, true)
		return true
	}
	if importData.Profile.Religion != "" && importData.Profile.Religion != currentData.HReligion {
		logger(1, "Religion: "+importData.Profile.Religion+" - "+currentData.HReligion, true)
		return true
	}
	if importData.Profile.HomeTelephone != "" && importData.Profile.HomeTelephone != currentData.HHomeTelephoneNumber {
		logger(1, "HomeTelephone: "+importData.Profile.HomeTelephone+" - "+currentData.HHomeTelephoneNumber, true)
		return true
	}
	if importData.Profile.SocialNetworkA != "" && importData.Profile.SocialNetworkA != currentData.HSnA {
		logger(1, "SocialNetworkA: "+importData.Profile.SocialNetworkA+" - "+currentData.HSnA, true)
		return true
	}
	if importData.Profile.SocialNetworkB != "" && importData.Profile.SocialNetworkB != currentData.HSnB {
		logger(1, "SocialNetworkB: "+importData.Profile.SocialNetworkB+" - "+currentData.HSnB, true)
		return true
	}
	if importData.Profile.SocialNetworkC != "" && importData.Profile.SocialNetworkC != currentData.HSnC {
		logger(1, "SocialNetworkC: "+importData.Profile.SocialNetworkC+" - "+currentData.HSnC, true)
		return true
	}
	if importData.Profile.SocialNetworkD != "" && importData.Profile.SocialNetworkD != currentData.HSnD {
		logger(1, "SocialNetworkD: "+importData.Profile.SocialNetworkD+" - "+currentData.HSnD, true)
		return true
	}
	if importData.Profile.SocialNetworkG != "" && importData.Profile.SocialNetworkG != currentData.HSnE {
		logger(1, "SocialNetworkE: "+importData.Profile.SocialNetworkE+" - "+currentData.HSnE, true)
		return true
	}
	if importData.Profile.SocialNetworkG != "" && importData.Profile.SocialNetworkG != currentData.HSnF {
		logger(1, "SocialNetworkF: "+importData.Profile.SocialNetworkF+" - "+currentData.HSnF, true)
		return true
	}
	if importData.Profile.SocialNetworkG != "" && importData.Profile.SocialNetworkG != currentData.HSnG {
		logger(1, "SocialNetworkG: "+importData.Profile.SocialNetworkG+" - "+currentData.HSnG, true)
		return true
	}
	if importData.Profile.SocialNetworkH != "" && importData.Profile.SocialNetworkH != currentData.HSnH {
		logger(1, "SocialNetworkH: "+importData.Profile.SocialNetworkH+" - "+currentData.HSnH, true)
		return true
	}
	if importData.Profile.PersonalInterests != "" && importData.Profile.PersonalInterests != currentData.HPersonalInterests {
		logger(1, "PersonalInterests: "+importData.Profile.PersonalInterests+" - "+currentData.HPersonalInterests, true)
		return true
	}
	if importData.Profile.HomeAddress != "" && importData.Profile.HomeAddress != currentData.HHomeAddress {
		logger(1, "HomeAddress: "+importData.Profile.HomeAddress+" - "+currentData.HHomeAddress, true)
		return true
	}
	if importData.Profile.PersonalBlog != "" && importData.Profile.PersonalBlog != currentData.HBlog {
		logger(1, "PersonalBlog: "+importData.Profile.PersonalBlog+" - "+currentData.HBlog, true)
		return true
	}
	if importData.Profile.Attrib1 != "" && importData.Profile.Attrib1 != currentData.HAttrib1 {
		logger(1, "Attrib1: "+importData.Profile.Attrib1+" - "+currentData.HAttrib1, true)
		return true
	}
	if importData.Profile.Attrib2 != "" && importData.Profile.Attrib2 != currentData.HAttrib2 {
		logger(1, "Attrib2: "+importData.Profile.Attrib2+" - "+currentData.HAttrib2, true)
		return true
	}
	if importData.Profile.Attrib3 != "" && importData.Profile.Attrib3 != currentData.HAttrib3 {
		logger(1, "Attrib3: "+importData.Profile.Attrib3+" - "+currentData.HAttrib3, true)
		return true
	}
	if importData.Profile.Attrib4 != "" && importData.Profile.Attrib4 != currentData.HAttrib4 {
		logger(1, "Attrib4: "+importData.Profile.Attrib4+" - "+currentData.HAttrib4, true)
		return true
	}
	if importData.Profile.Attrib5 != "" && importData.Profile.Attrib5 != currentData.HAttrib5 {
		logger(1, "Attrib5: "+importData.Profile.Attrib5+" - "+currentData.HAttrib5, true)
		return true
	}
	if importData.Profile.Attrib6 != "" && importData.Profile.Attrib6 != currentData.HAttrib6 {
		logger(1, "Attrib6: "+importData.Profile.Attrib6+" - "+currentData.HAttrib6, true)
		return true
	}
	if importData.Profile.Attrib7 != "" && importData.Profile.Attrib7 != currentData.HAttrib7 {
		logger(1, "Attrib7: "+importData.Profile.Attrib7+" - "+currentData.HAttrib7, true)
		return true
	}
	if importData.Profile.Attrib8 != "" && importData.Profile.Attrib8 != currentData.HAttrib8 {
		logger(1, "Attrib8: "+importData.Profile.Attrib8+" - "+currentData.HAttrib8, true)
		return true
	}
	return false
}

//-- For Each Import Actions process the data
//func processImportActions(l *ldap.Entry) string {
func processImportActions(l *map[string]interface{}) string {

	//-- Set User Account Attributes
	var data = new(userWorkingDataStruct)
	data.DB = l
	//-- init map
	data.Custom = make(map[string]string)
	data.Account.UserID = getUserFieldValue(l, "UserID", data.Custom)

	logger(1, "Post Import Actions for: "+data.Account.UserID, false)
	//-- Loop Matches
	for _, action := range AzureImportConf.Actions {
		switch action.Action {
		case "Regex":
			//-- Grab value from DB
			Outcome := processComplexField(l, action.Value)
			//-- Grab Value from Existing Custom Feild
			Outcome = processImportAction(data.Custom, Outcome)
			//-- Process Regex
			Outcome = processRegexOnString(action.Options.RegexValue, Outcome)
			//-- Store
			data.Custom["{"+action.Output+"}"] = Outcome

			logger(1, "Regex Output: "+Outcome, false)
		case "Replace":
			//-- Grab value from DB
			Outcome := processComplexField(l, action.Value)
			//-- Grab Value from Existing Custom Feild
			Outcome = processImportAction(data.Custom, Outcome)
			//-- Run Replace
			Outcome = strings.Replace(Outcome, action.Options.ReplaceFrom, action.Options.ReplaceWith, -1)
			//-- Store
			data.Custom["{"+action.Output+"}"] = Outcome

			logger(1, "Replace Output: "+Outcome, false)
		case "Trim":
			//-- Grab value from DB
			Outcome := processComplexField(l, action.Value)
			//-- Grab Value from Existing Custom Feild
			Outcome = processImportAction(data.Custom, Outcome)
			//-- Run Replace
			Outcome = strings.TrimSpace(Outcome)
			Outcome = strings.Replace(Outcome, "\n", "", -1)
			Outcome = strings.Replace(Outcome, "\r", "", -1)
			Outcome = strings.Replace(Outcome, "\r\n", "", -1)
			//-- Store
			data.Custom["{"+action.Output+"}"] = Outcome

			logger(1, "Trim Output: "+Outcome, false)
		case "None":
			//-- Grab value
			Outcome := processComplexField(l, action.Value)
			//-- Grab Value from Existing Custom Feild
			Outcome = processImportAction(data.Custom, Outcome)
			//-- Store
			data.Custom["{"+action.Output+"}"] = Outcome

			logger(1, "Copy Output: "+Outcome, false)

		default:
			logger(1, "Unknown Action: "+action.Action, false)
		}
	}
	//-- Store Result in map of userid
	var userID = strings.ToLower(data.Account.UserID)
	HornbillCache.UsersWorking[userID] = data
	return userID
}

//-- For Each DB User Process Account And Mappings
//func processUserParams(l *ldap.Entry, userID string) {
func processUserParams(l *map[string]interface{}, userID string) {

	data := HornbillCache.UsersWorking[userID]
	data.Account.LoginID = getUserFieldValue(l, "LoginID", data.Custom)
	data.Account.EmployeeID = getUserFieldValue(l, "EmployeeID", data.Custom)
	data.Account.UserType = getUserFieldValue(l, "UserType", data.Custom)
	data.Account.Name = getUserFieldValue(l, "Name", data.Custom)
	data.Account.Password = getUserFieldValue(l, "Password", data.Custom)
	data.Account.FirstName = getUserFieldValue(l, "FirstName", data.Custom)
	data.Account.LastName = getUserFieldValue(l, "LastName", data.Custom)
	data.Account.JobTitle = getUserFieldValue(l, "JobTitle", data.Custom)
	data.Account.Site = getUserFieldValue(l, "Site", data.Custom)
	data.Account.Phone = getUserFieldValue(l, "Phone", data.Custom)
	data.Account.Email = getUserFieldValue(l, "Email", data.Custom)
	data.Account.Mobile = getUserFieldValue(l, "Mobile", data.Custom)
	data.Account.AbsenceMessage = getUserFieldValue(l, "AbsenceMessage", data.Custom)
	data.Account.TimeZone = getUserFieldValue(l, "TimeZone", data.Custom)
	data.Account.Language = getUserFieldValue(l, "Language", data.Custom)
	data.Account.DateTimeFormat = getUserFieldValue(l, "DateTimeFormat", data.Custom)
	data.Account.DateFormat = getUserFieldValue(l, "DateFormat", data.Custom)
	data.Account.TimeFormat = getUserFieldValue(l, "TimeFormat", data.Custom)
	data.Account.CurrencySymbol = getUserFieldValue(l, "CurrencySymbol", data.Custom)
	data.Account.CountryCode = getUserFieldValue(l, "CountryCode", data.Custom)

	data.Profile.MiddleName = getProfileFieldValue(l, "MiddleName", data.Custom)
	data.Profile.JobDescription = getProfileFieldValue(l, "JobDescription", data.Custom)
	data.Profile.Manager = getProfileFieldValue(l, "Manager", data.Custom)
	data.Profile.WorkPhone = getProfileFieldValue(l, "WorkPhone", data.Custom)
	data.Profile.Qualifications = getProfileFieldValue(l, "Qualifications", data.Custom)
	data.Profile.Interests = getProfileFieldValue(l, "Interests", data.Custom)
	data.Profile.Expertise = getProfileFieldValue(l, "Expertise", data.Custom)
	data.Profile.Gender = getProfileFieldValue(l, "Gender", data.Custom)
	data.Profile.Dob = getProfileFieldValue(l, "Dob", data.Custom)
	data.Profile.Nationality = getProfileFieldValue(l, "Nationality", data.Custom)
	data.Profile.Religion = getProfileFieldValue(l, "Religion", data.Custom)
	data.Profile.HomeTelephone = getProfileFieldValue(l, "HomeTelephone", data.Custom)
	data.Profile.SocialNetworkA = getProfileFieldValue(l, "SocialNetworkA", data.Custom)
	data.Profile.SocialNetworkB = getProfileFieldValue(l, "SocialNetworkB", data.Custom)
	data.Profile.SocialNetworkC = getProfileFieldValue(l, "SocialNetworkC", data.Custom)
	data.Profile.SocialNetworkD = getProfileFieldValue(l, "SocialNetworkD", data.Custom)
	data.Profile.SocialNetworkE = getProfileFieldValue(l, "SocialNetworkE", data.Custom)
	data.Profile.SocialNetworkF = getProfileFieldValue(l, "SocialNetworkF", data.Custom)
	data.Profile.SocialNetworkG = getProfileFieldValue(l, "SocialNetworkG", data.Custom)
	data.Profile.SocialNetworkH = getProfileFieldValue(l, "SocialNetworkH", data.Custom)
	data.Profile.PersonalInterests = getProfileFieldValue(l, "PersonalInterests", data.Custom)
	data.Profile.HomeAddress = getProfileFieldValue(l, "HomeAddress", data.Custom)
	data.Profile.PersonalBlog = getProfileFieldValue(l, "PersonalBlog", data.Custom)
	data.Profile.Attrib1 = getProfileFieldValue(l, "Attrib1", data.Custom)
	data.Profile.Attrib2 = getProfileFieldValue(l, "Attrib2", data.Custom)
	data.Profile.Attrib3 = getProfileFieldValue(l, "Attrib3", data.Custom)
	data.Profile.Attrib4 = getProfileFieldValue(l, "Attrib4", data.Custom)
	data.Profile.Attrib5 = getProfileFieldValue(l, "Attrib5", data.Custom)
	data.Profile.Attrib6 = getProfileFieldValue(l, "Attrib6", data.Custom)
	data.Profile.Attrib7 = getProfileFieldValue(l, "Attrib7", data.Custom)
	data.Profile.Attrib8 = getProfileFieldValue(l, "Attrib8", data.Custom)
}
