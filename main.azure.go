package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//getBearerToken - gets and returns a bearer token for Azure authentication, or the current session bearer token if not expired
func getBearerToken() (string, error) {
	currentEpoch := time.Now().Unix()
	if globalBearerToken != "" && ((currentEpoch + 200) < globalTokenExpiry) {
		logger(1, "[SCRIPT] Re-using BearerToken", false)
		return globalBearerToken, nil
	}

	logger(1, "[SCRIPT] Generating Bearer Token", false)
	strClientID := AzureImportConf.AzureConf.ClientID
	strClientSecret := AzureImportConf.AzureConf.ClientSecret
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_secret", strClientSecret)
	data.Set("client_id", strClientID)
	data.Set("resource", "https://graph.windows.net")
	strData := data.Encode()
	strTentant := AzureImportConf.AzureConf.Tenant
	strURL := "https://login.microsoftonline.com/" + strTentant + "/oauth2/token"

	var xmlmcstr = []byte(strData)
	req, err := http.NewRequest("POST", strURL, bytes.NewBuffer(xmlmcstr))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	req.Header.Set("Accept", "text/json")
	duration := time.Second * time.Duration(30)

	client := &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}, Timeout: duration}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("Cant read the body of the response")
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)

	if qerr != nil {
		return "", errors.New("Cant read the JSON")
	}

	q := f.(map[string]interface{})
	strBearerToken := q["access_token"].(string)
	globalBearerToken = strBearerToken
	globalTokenExpiry, _ = strconv.ParseInt(q["expires_on"].(string), 10, 64)
	logger(1, "[SCRIPT] Got New BearerToken", false)
	return strBearerToken, nil
}

//queryUsers -- Query Graph API for Users of current type
//-- Builds map of users, returns true if successful
func queryUsers() (bool, []map[string]interface{}) {
	//Clear existing User Map down
	var ArrUserMaps []map[string]interface{}

	logger(2, "[Azure] Query Azure Data using Graph API. Please wait...", true)

	strBearerToken, err := getBearerToken()
	if err != nil || strBearerToken == "" {
		logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrUserMaps
	}

	strTenant := AzureImportConf.AzureConf.Tenant
	strURL := "https://graph.windows.net/" + strTenant + "/users?"

	data := url.Values{}
	data.Set("api-version", AzureImportConf.AzureConf.APIVersion)
	strFilter := AzureImportConf.AzureConf.UserFilter
	if strFilter != "" {
		data.Set("$filter", strFilter)
	}
	strData := data.Encode()
	strURL += strData
	logger(1, "[AZURE] API URL: "+strURL, false)
	req, err := http.NewRequest("GET", strURL, nil) //, bytes.NewBuffer(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	req.Header.Set("Authorization", "Bearer "+strBearerToken)
	req.Header.Set("Accept", "application/json")
	duration := time.Second * time.Duration(30)

	client := &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}, Timeout: duration}
	resp, err := client.Do(req)
	if err != nil {
		logger(4, " [Azure] Connection Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrUserMaps
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		logger(4, " [Azure] Error: "+fmt.Sprintf("%v", err), true)
		logger(4, " [Azure] Response: "+bodyString, true)
		return false, ArrUserMaps
	}
	logger(2, "[Azure] Connection Successful", false)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger(4, " [Azure] Cannot read the body of the response", true)
		return false, ArrUserMaps
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)
	if qerr != nil {
		logger(4, " [Azure] Cannot read the JSON", true)
		return false, ArrUserMaps
	}

	//Build map of users
	intUserCount := 0

	q := f.(map[string]interface{})
	if aResults, ok := q["value"].([]interface{}); ok {

		for _, userDetails := range aResults {
			intUserCount++

			blubber := userDetails.(map[string]interface{})
			ArrUserMaps = append(ArrUserMaps, blubber)
		}
	}

	logger(2, fmt.Sprintf("[Azure] Found %d results", intUserCount), false)
	return true, ArrUserMaps
}

//queryGroup -- Query Graph API for User objects within a Group
//-- Builds map of users, returns true if successful
func queryGroup(groupID string) (bool, []map[string]interface{}) {
	//Clear existing User Map down
	var ArrUserMaps []map[string]interface{}
	logger(2, "[Azure] Query Azure Data using Graph API. Please wait...", true)

	strBearerToken, err := getBearerToken()

	if err != nil || strBearerToken == "" {
		logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrUserMaps
	}

	strTenant := AzureImportConf.AzureConf.Tenant
	strURL := "https://graph.windows.net/" + strTenant + "/groups/" + groupID + "/$links/members?api-version=" + AzureImportConf.AzureConf.APIVersion

	logger(1, "[AZURE] API URL: "+strURL, false)
	req, err := http.NewRequest("GET", strURL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	req.Header.Set("Authorization", "Bearer "+strBearerToken)
	req.Header.Set("Accept", "application/json")
	duration := time.Second * time.Duration(30)

	client := &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}, Timeout: duration}
	resp, err := client.Do(req)
	if err != nil {
		logger(4, " [Azure] Connection Error: "+fmt.Sprintf("%v", err), true)
		return false, ArrUserMaps
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		io.Copy(ioutil.Discard, resp.Body)
		logger(4, " [Azure] Error: "+fmt.Sprintf("%v", err), true)
		logger(4, " [Azure] Response: "+bodyString, true)
		return false, ArrUserMaps
	}
	logger(2, "[Azure] Connection Successful", false)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger(4, " [Azure] Cannot read the body of the response", true)
		return false, ArrUserMaps
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)
	if qerr != nil {
		logger(4, " [Azure] Cannot read the JSON", true)
		return false, ArrUserMaps
	}

	//Build map full of users
	intUserCount := 0
	q := f.(map[string]interface{})
	if aResults, ok := q["value"].([]interface{}); ok {
		//Return the API URL for each user record found in the group
		for _, userDetails := range aResults {
			userURL, ok := userDetails.(map[string]interface{})
			if ok {
				//Now go get user details and add to map
				if strUserURL, urlOK := userURL["url"].(string); urlOK {
					if strings.Contains(strUserURL, "Microsoft.DirectoryServices.User") {

						strCurrBearerToken, err := getBearerToken()
						if err != nil || strCurrBearerToken == "" {
							logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
							return false, ArrUserMaps
						}

						intUserCount++
						strUserURL += "?api-version=" + AzureImportConf.AzureConf.APIVersion
						logger(1, "[AZURE] User API URL: "+strUserURL, false)
						req, err := http.NewRequest("GET", strUserURL, nil)
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("User-Agent", "Go-http-client/1.1")
						req.Header.Set("Authorization", "Bearer "+strCurrBearerToken)
						req.Header.Set("Accept", "application/json")
						duration := time.Second * time.Duration(30)

						clientUser := &http.Client{Transport: &http.Transport{
							Proxy: http.ProxyFromEnvironment,
						}, Timeout: duration}
						respUser, err := clientUser.Do(req)
						if err != nil {
							logger(4, " [Azure] Connection Error: "+fmt.Sprintf("%v", err), true)
							return false, ArrUserMaps
						}
						defer respUser.Body.Close()
						//-- Check for HTTP Response
						if respUser.StatusCode != 200 {
							errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
							err = errors.New(errorString)
							//Drain the body so we can reuse the connection
							bodyBytes, _ := ioutil.ReadAll(respUser.Body)
							bodyString := string(bodyBytes)
							io.Copy(ioutil.Discard, respUser.Body)
							logger(4, " [Azure] Error: "+fmt.Sprintf("%v", err), true)
							logger(4, " [Azure] Response: "+bodyString, true)
							return false, ArrUserMaps
						}
						logger(2, "[Azure] Connection Successful", false)

						userBody, err := ioutil.ReadAll(respUser.Body)
						if err != nil {
							logger(4, " [Azure] Cannot read the body of the response", true)
							return false, ArrUserMaps
						}

						var fuser interface{}
						qerror := json.Unmarshal(userBody, &fuser)
						if qerror != nil {
							logger(4, " [Azure] Cannot read the JSON", true)
							return false, ArrUserMaps
						}
						blubber := fuser.(map[string]interface{})
						ArrUserMaps = append(ArrUserMaps, blubber)
					}
				}
			}
		}
	}

	logger(2, fmt.Sprintf("[Azure] Found %d Users", intUserCount), true)
	if intUserCount > 0 {
		return true, ArrUserMaps
	}
	return false, ArrUserMaps
}
