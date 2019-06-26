package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
	hornbillpasswordgen "github.com/hornbill/goHornbillPasswordGen"
)

//-- Generate Password String
func generatePasswordString(userID string, mustNotContain []string) string {
	pwdinst := hornbillpasswordgen.NewPasswordInstance()
	pwdinst.Length = passwordProfile.Length
	pwdinst.UseLower = true
	pwdinst.ForceLower = passwordProfile.ForceLower
	pwdinst.UseNumeric = true
	pwdinst.ForceNumeric = passwordProfile.ForceNumeric
	pwdinst.UseUpper = true
	pwdinst.ForceUpper = passwordProfile.ForceUpper
	pwdinst.UseSpecial = true
	pwdinst.ForceSpecial = passwordProfile.ForceSpecial
	pwdinst.Blacklist = passwordProfile.Blacklist
	if passwordProfile.CheckMustNotContain {
		pwdinst.MustNotContain = mustNotContain
	}

	//Generate a new password
	newPassword, err := pwdinst.GenPassword()

	if err != nil {
		logger(4, "Failed Password Auto Generation for: "+userID+"  "+fmt.Sprintf("%v", err), false)
		return ""
	}
	return newPassword
}

//getPasswordProfile - retrieves the user password profile settings from your Hornbill instance, applies ready for the password generator to use
func getPasswordProfile() {
	mc := apiLib.NewXmlmcInstance(AzureImportConf.InstanceID)
	mc.SetAPIKey(AzureImportConf.APIKey)
	mc.SetJSONResponse(true)
	mc.SetParam("filter", "security.user.passwordPolicy")
	RespBody, xmlmcErr := mc.Invoke("admin", "sysOptionGet")
	var JSONResp xmlmcSettingResponse
	if xmlmcErr != nil {
		logger(4, "Unable to run sysOptionGet "+fmt.Sprintf("%s", xmlmcErr), false)
		return
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to unmarshal sysOptionGet response "+fmt.Sprintf("%s", err), false)
		return
	}
	if JSONResp.State.ErrorRet != "" {
		logger(4, "Error returned from sysOptionGet "+JSONResp.State.ErrorRet, false)
		return
	}
	//Process Password Profile
	//--Work through profile settings
	for _, val := range JSONResp.Params.Option {
		switch val.Key {
		case "security.user.passwordPolicy.checkBlacklists":
			passwordProfile.Blacklist = processBlacklists()
		case "security.user.passwordPolicy.checkPersonalInfo":
			passwordProfile.CheckMustNotContain, _ = strconv.ParseBool(val.Value)
		case "security.user.passwordPolicy.minimumLength":
			if val.Value == "0" {
				passwordProfile.Length = defaultPasswordLength
			} else {
				passwordProfile.Length, _ = strconv.Atoi(val.Value)
			}
		case "security.user.passwordPolicy.mustContainLowerCase":
			passwordProfile.ForceLower, _ = strconv.Atoi(val.Value)
		case "security.user.passwordPolicy.mustContainNumeric":
			passwordProfile.ForceNumeric, _ = strconv.Atoi(val.Value)
		case "security.user.passwordPolicy.mustContainSpecial":
			passwordProfile.ForceSpecial, _ = strconv.Atoi(val.Value)
		case "security.user.passwordPolicy.mustContainUpperCase":
			passwordProfile.ForceUpper, _ = strconv.Atoi(val.Value)
		}
	}
	totalForce := passwordProfile.ForceLower + passwordProfile.ForceNumeric + passwordProfile.ForceSpecial + passwordProfile.ForceUpper
	if passwordProfile.Length < totalForce {
		passwordProfile.Length = totalForce
	}
}

func processBlacklists() []string {
	var blacklist []string
	for _, v := range blacklistURLs {
		blacklistContent := getBlacklist(v)
		for _, l := range blacklistContent {
			alreadyInList := false
			for _, m := range blacklist {
				if strings.EqualFold(m, l) {
					alreadyInList = true
				}
			}
			if !alreadyInList {
				blacklist = append(blacklist, l)
			}
		}
	}
	return blacklist
}

func getBlacklist(blacklistURL string) []string {
	var blacklist []string
	//-- Get JSON Config
	response, err := http.Get(blacklistURL)
	if err != nil || response.StatusCode != 200 {
		logger(4, "Unexpected status "+strconv.Itoa(response.StatusCode)+" returned from "+blacklistURL, false)
		return blacklist
	}
	//-- Close Connection
	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	if err := scanner.Err(); err != nil {
		logger(4, "Unable to decode blacklist from "+blacklistURL+": "+fmt.Sprintf("%v", err), false)
		return blacklist
	}
	for scanner.Scan() {
		textRow := scanner.Text()
		trimmedRow := strings.TrimSpace(textRow)
		//Ignore comment
		if string([]rune(trimmedRow)[0]) != "#" {
			blacklist = append(blacklist, trimmedRow)
		}
	}

	return blacklist
}
