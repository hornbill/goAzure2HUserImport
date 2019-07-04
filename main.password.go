package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
	hornbillpasswordgen "github.com/hornbill/goHornbillPasswordGen"
)

//-- Generate Password String
func generatePasswordString(userID string, mustNotContain []string, buffer *bytes.Buffer) string {
	pwdinst := hornbillpasswordgen.NewPasswordInstance()
	if configDebug {
		pwdinst.SetDebug()
	}
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
	newPassword, debug, err := pwdinst.GenPassword()

	if err != nil {
		buffer.WriteString(loggerGen(4, "Failed Password Auto Generation for: "+userID+"  "+fmt.Sprintf("%v", err), true))
		return ""
	}
	if configDebug && len(debug) > 0 {
		buffer.WriteString(loggerGen(1, "[PASSWORD] Debugging information from password generator:", true))
		for _, v := range debug {
			buffer.WriteString(loggerGen(1, "[PASSWORD] "+v, true))
		}
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
			checkMustNotContain, _ := strconv.ParseBool(val.Value)
			if checkMustNotContain {
				passwordProfile.Blacklist = processBlacklists()
			}
		case "security.user.passwordPolicy.checkPersonalInfo":
			passwordProfile.CheckMustNotContain, err = strconv.ParseBool(val.Value)
			if err != nil {
				passwordProfile.CheckMustNotContain = false
				if configDebug {
					logger(3, "Unable to read password policy checkPersonalInfo value: "+err.Error(), false)
				}
			}
		case "security.user.passwordPolicy.minimumLength":
			if val.Value == "0" {
				passwordProfile.Length = defaultPasswordLength
			} else {
				passwordProfile.Length, err = strconv.Atoi(val.Value)
				if err != nil {
					passwordProfile.Length = defaultPasswordLength
					if configDebug {
						logger(3, "Unable to read password policy length, setting to default of "+strconv.Itoa(defaultPasswordLength)+": "+err.Error(), false)
					}
				}
			}
		case "security.user.passwordPolicy.mustContainLowerCase":
			passwordProfile.ForceLower, err = strconv.Atoi(val.Value)
			if err != nil && configDebug {
				logger(3, "Unable to read password policy mustContainLowerCase value: "+err.Error(), false)
			}
		case "security.user.passwordPolicy.mustContainNumeric":
			passwordProfile.ForceNumeric, err = strconv.Atoi(val.Value)
			if err != nil && configDebug {
				logger(3, "Unable to read password policy mustContainNumeric value: "+err.Error(), false)
			}
		case "security.user.passwordPolicy.mustContainSpecial":
			passwordProfile.ForceSpecial, err = strconv.Atoi(val.Value)
			if err != nil && configDebug {
				logger(3, "Unable to read password policy mustContainSpecial value: "+err.Error(), false)
			}
		case "security.user.passwordPolicy.mustContainUpperCase":
			passwordProfile.ForceUpper, err = strconv.Atoi(val.Value)
			if err != nil && configDebug {
				logger(3, "Unable to read password policy mustContainUpperCase value: "+err.Error(), false)
			}
		}
	}
	if passwordProfile.Length == 0 {
		passwordProfile.Length = defaultPasswordLength
		if configDebug {
			logger(3, "Password policy length set to 0 after reading policy, setting to default of "+strconv.Itoa(defaultPasswordLength)+": ", false)
		}
	}

	totalForce := passwordProfile.ForceLower + passwordProfile.ForceNumeric + passwordProfile.ForceSpecial + passwordProfile.ForceUpper
	if passwordProfile.Length < totalForce {
		passwordProfile.Length = totalForce
	}
	if passwordProfile.Length == 0 {
		passwordProfile.Length = defaultPasswordLength
		if configDebug {
			logger(3, "Password policy length STILL set to 0 after checking force values in policy, setting to default of "+strconv.Itoa(defaultPasswordLength)+": ", false)
		}
	}

	if configDebug {
		logger(1, "Password Profile", false)
		blacklist := strings.Join(passwordProfile.Blacklist, ", ")
		logger(1, "checkBlacklists: "+blacklist, false)
		logger(1, "checkPersonalInfo: "+strconv.FormatBool(passwordProfile.CheckMustNotContain), false)
		logger(1, "minimumLength: "+strconv.Itoa(passwordProfile.Length), false)
		logger(1, "mustContainLowerCase: "+strconv.Itoa(passwordProfile.ForceLower), false)
		logger(1, "mustContainNumeric: "+strconv.Itoa(passwordProfile.ForceNumeric), false)
		logger(1, "mustContainSpecial: "+strconv.Itoa(passwordProfile.ForceSpecial), false)
		logger(1, "mustContainUpperCase: "+strconv.Itoa(passwordProfile.ForceUpper), false)
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
