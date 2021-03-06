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
	"regexp"
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
	strClientID := azureImportConf.AzureConf.ClientID
	strClientSecret := azureImportConf.AzureConf.ClientSecret
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_secret", strClientSecret)
	data.Set("client_id", strClientID)
	data.Set("resource", apiResource)
	strData := data.Encode()
	strTentant := azureImportConf.AzureConf.Tenant
	strURL := "https://login.microsoftonline.com/" + strTentant + "/oauth2/token"

	var xmlmcstr = []byte(strData)
	req, _ := http.NewRequest("POST", strURL, bytes.NewBuffer(xmlmcstr))
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
		return "", errors.New("cant read the body of the response")
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)

	if qerr != nil {
		return "", errors.New("cant read the JSON")
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
func queryAzure() bool {
	//Clear existing User Map down
	var ArrUserMaps []map[string]interface{}

	logger(2, "[Azure] Query Azure Data using Graph API. Please wait...", true)

	strBearerToken, err := getBearerToken()
	if err != nil || strBearerToken == "" {
		logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
		return false
	}

	strURL := apiResource + "/" + azureImportConf.AzureConf.APIVersion + "/users?" //$top=1&"
	if strAzurePagerToken != "" {
		strURL += "$skiptoken=" + strAzurePagerToken + "&"
	}

	data := url.Values{}
	strFilter := azureImportConf.AzureConf.UserFilter
	if strFilter != "" {
		data.Set("$filter", strFilter)
	}
	//Add user properties to search
	if len(azureImportConf.AzureConf.UserProperties) > 0 {
		var searchProperties = []string{"id", "businessPhones", "displayName", "givenName", "jobTitle", "mail", "mobilePhone", "officeLocation", "preferredLanguage", "surname", "userPrincipalName"}
		searchProperties = append(searchProperties, azureImportConf.AzureConf.UserProperties...)
		data.Set("$select", strings.Join(searchProperties, ","))
	}
	strData := data.Encode()
	strURL += strData
	logger(1, "[AZURE] API URL: "+strURL, false)
	req, _ := http.NewRequest("GET", strURL, nil) //, bytes.NewBuffer(""))
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
		return false
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
		return false
	}
	logger(2, "[Azure] Connection Successful", false)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger(4, " [Azure] Cannot read the body of the response", true)
		return false
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)
	if qerr != nil {
		logger(4, " [Azure] Cannot read the JSON", true)
		return false
	}

	//Build map of users
	intUserCount := 0

	q := f.(map[string]interface{})
	if aResults, ok := q["value"].([]interface{}); ok {

		for _, userDetails := range aResults {
			intUserCount++

			blubber := userDetails.(map[string]interface{})

			bln, manager := getAzureManager(blubber["id"].(string))
			if bln {
				blubber["manager"] = manager
			}
			ArrUserMaps = append(ArrUserMaps, blubber)
		}
	}
	if strNextLink, ok := q["@odata.nextLink"]; ok {
		arrNewPagerToken := strings.SplitAfter(strNextLink.(string), "skiptoken=")
		strTokenToTidy := strings.SplitAfter(arrNewPagerToken[1], "&")
		logger(1, " [Azure] Determined Token: "+strTokenToTidy[0], false)
		strAzurePagerToken = strTokenToTidy[0]
	} else {
		logger(1, " [Azure] No Skip Token Found", false)
		strAzurePagerToken = ""
	}
	logger(2, fmt.Sprintf("[Azure] Found %d results", intUserCount), false)
	localAzureUsers = append(localAzureUsers, ArrUserMaps...)
	return true
}

//queryGroup -- Query Graph API for User objects within a Group
//-- Builds map of users, returns true if successful
func queryGroup(groupID string) bool {
	//Clear existing User Map down
	var ArrUserMaps []map[string]interface{}
	logger(2, "[Azure] Query Azure Data using Graph API. Please wait...", true)

	strBearerToken, err := getBearerToken()

	if err != nil || strBearerToken == "" {
		logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
		return false
	}

	strURL := apiResource + "/" + azureImportConf.AzureConf.APIVersion + "/groups/" + groupID + "/members?"
	if strAzurePagerToken != "" {
		strURL += "&$skiptoken=" + strAzurePagerToken
	}
	logger(1, "[AZURE] API URL: "+strURL, false)
	req, _ := http.NewRequest("GET", strURL, nil)
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
		return false
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		io.Copy(ioutil.Discard, resp.Body)
		logger(4, " [Azure] Error: "+fmt.Sprintf("%v", err), true)
		logger(4, " [Azure] Response: "+bodyString, true)
		return false
	} else if resp.StatusCode == 404 {

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		logger(4, " [Azure] Response: "+bodyString, false)

		//Drain the body so we can reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		logger(1, " [Azure] Response: No entries found for "+groupID, true)
		return false
	}
	logger(2, "[Azure] Connection Successful", false)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger(4, " [Azure] Cannot read the body of the response", true)
		return false
	}

	var f interface{}
	qerr := json.Unmarshal(body, &f)
	if qerr != nil {
		logger(4, " [Azure] Cannot read the JSON", true)
		return false
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

				if strUserURL, urlOK := userURL["@odata.type"].(string); urlOK {
					if strings.Contains(strUserURL, "microsoft.graph.user") {

						strCurrBearerToken, err := getBearerToken()
						if err != nil || strCurrBearerToken == "" {
							logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
							return false
						}

						intUserCount++
						strUserURL = apiResource + "/" + azureImportConf.AzureConf.APIVersion + "/users/" + userURL["id"].(string)

						logger(1, "[AZURE] User API URL: "+strUserURL, false)
						//Add user properties to search
						data := url.Values{}
						if len(azureImportConf.AzureConf.UserProperties) > 0 {
							var searchProperties = []string{"id", "businessPhones", "displayName", "givenName", "jobTitle", "mail", "mobilePhone", "officeLocation", "preferredLanguage", "surname", "userPrincipalName"}
							searchProperties = append(searchProperties, azureImportConf.AzureConf.UserProperties...)
							data.Set("$select", strings.Join(searchProperties, ","))
							strData := data.Encode()
							strUserURL += "?" + strData
						}
						req, _ := http.NewRequest("GET", strUserURL, nil)
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
							return false
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
							return false
						}
						logger(2, "[Azure] Connection Successful", false)

						userBody, err := ioutil.ReadAll(respUser.Body)
						if err != nil {
							logger(4, " [Azure] Cannot read the body of the response", true)
							return false
						}

						var fuser interface{}
						qerror := json.Unmarshal(userBody, &fuser)
						if qerror != nil {
							logger(4, " [Azure] Cannot read the JSON", true)
							return false
						}
						blubber := fuser.(map[string]interface{})

						bln, manager := getAzureManager(blubber["userPrincipalName"].(string))
						if bln {
							blubber["manager"] = manager
						}

						ArrUserMaps = append(ArrUserMaps, blubber)
					}
				}
			}
		}
	}

	if strNextLink, ok := q["@odata.nextLink"]; ok {
		arrNewPagerToken := strings.SplitAfter(strNextLink.(string), "skiptoken=")
		strTokenToTidy := strings.SplitAfter(arrNewPagerToken[1], "&")
		logger(1, " [Azure] Determined Token: "+strTokenToTidy[0], false)
		strAzurePagerToken = strTokenToTidy[0]
	} else {
		logger(1, " [Azure] No Skip Token Found", false)
		strAzurePagerToken = ""
	}

	logger(2, fmt.Sprintf("[Azure] Found %d Users", intUserCount), true)
	if intUserCount > 0 {
		localAzureUsers = append(localAzureUsers, ArrUserMaps...)
		return true
	}

	return false
}

func getAzureManager(userPrincipalName string) (bool, string) {
	logger(1, "[Azure] Querying Azure Manager. Please wait...", false)

	strBearerToken, err := getBearerToken()
	if err != nil || strBearerToken == "" {
		logger(4, " [Azure] BearerToken Error: "+fmt.Sprintf("%v", err), true)
		return false, ""
	}

	strURL := apiResource + "/" + azureImportConf.AzureConf.APIVersion + "/users/" + userPrincipalName + "/manager" //?" //$top=1&"
	data := url.Values{}
	strData := data.Encode()
	strURL += strData
	logger(1, "[AZURE] API URL: "+strURL, false)
	req, _ := http.NewRequest("GET", strURL, nil)
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
		return false, ""
	}
	defer resp.Body.Close()

	//-- Check for HTTP Response
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		errorString := fmt.Sprintf("Invalid HTTP Response: %d", resp.StatusCode)
		err = errors.New(errorString)
		//Drain the body so we can reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		logger(4, " [Azure] Error: "+fmt.Sprintf("%v", err), true)
		logger(4, " [Azure] Response: "+bodyString, true)
		return false, ""
	} else if resp.StatusCode == 404 {
		//Drain the body so we can reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		logger(1, " [Azure] Response: No manager found for "+userPrincipalName, true)
		return false, ""
	}
	logger(2, "[Azure] Connection Successful", false)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger(4, " [Azure] Cannot read the body of the response", true)
		return false, ""
	}
	var f interface{}
	qerr := json.Unmarshal(body, &f)
	if qerr != nil {
		logger(4, " [Azure] Cannot read the JSON", true)
		return false, ""
	}
	q := f.(map[string]interface{})
	strUserURL := ""
	if strUserURL, urlOK := q[azureImportConf.AzureConf.UserID].(string); urlOK {
		if strUserURL != "" {
			return true, strUserURL
		}
	}
	logger(2, fmt.Sprintf("[Azure] Found %s results", strUserURL), false)
	return true, ""
}

func read_azure_string_collection(text string, index_input ...int) string {
	//default index = 0
	index := 0
	if len(index_input) > 0 {
		index = index_input[0]
	}
	if azureImportConf.AzureConf.StringCollectionTweak {
		r, _ := regexp.Compile("\"?([^\",])*\"?")
		t := r.FindAllString(text, -1) //[index]
		e := ""
		if len(t) >= index {
			e = t[index]
		}
		e = strings.Trim(e, `"`)
		e = strings.Trim(e, `[`)
		e = strings.Trim(e, `]`)
		e = strings.Trim(e, `"`)
		return e
	} else {
		return text
	}
}
