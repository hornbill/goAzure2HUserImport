package main

import (
	"bytes"
	"fmt"

	"github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

func worker(id int, jobs <-chan int, results chan<- int, bar *pb.ProgressBar) {

	hIF := apiLib.NewXmlmcInstance(Flags.configInstanceID)
	hIF.SetAPIKey(Flags.configAPIKey)
	hIF.SetTimeout(Flags.configAPITimeout)
	hIF.SetJSONResponse(true)

	for index := range jobs {
		//-- Load Current User
		currentUser := HornbillCache.UsersWorkingIndex[index-1]
		//-- Buffer for Logging
		var buffer bytes.Buffer
		if currentUser.Jobs.create {
			createUser(hIF, currentUser, &buffer)
		}

		if currentUser.Jobs.update || currentUser.Jobs.updateType || currentUser.Jobs.updateSite {
			updateUser(hIF, currentUser, &buffer)
		}
		if currentUser.Jobs.updateProfile {
			updateUserProfile(hIF, currentUser, &buffer)
		}
		if currentUser.Jobs.updateImage {
			updateUserImage(hIF, currentUser, &buffer)
		}
		if len(currentUser.GroupsToRemove) > 0 {
			removeUserGroups(hIF, currentUser, &buffer)
		}
		if len(currentUser.Groups) > 0 {
			updateUserGroups(hIF, currentUser, &buffer)
		}
		if len(currentUser.Roles) > 0 {
			updateUserRoles(hIF, currentUser, &buffer)
		}
		if currentUser.Jobs.updateStatus {
			updateUserStatus(hIF, currentUser, &buffer)
		}
		if currentUser.Jobs.updateHomeOrg {
			userGroupSetHomeOrg(hIF, currentUser, &buffer)
		}
		bar.Increment()

		bufferMutex.Lock()
		loggerWriteBuffer(buffer.String())
		bufferMutex.Unlock()
		buffer.Reset()

		//-- Results
		results <- index * 2
	}

	mutexCounters.Lock()
	counters.traffic += hIF.GetCount()
	mutexCounters.Unlock()
}

func finaliseData() {
	logger(1, "Finalising User Data", true)

	//-- Current User data is in a map of userId to object we need to put this in an index based map
	HornbillCache.UsersWorkingIndex = make(map[int]*userWorkingDataStruct)
	var count int
	for userID := range HornbillCache.UsersWorking {
		HornbillCache.UsersWorkingIndex[count] = HornbillCache.UsersWorking[userID]
		count++
	}
	//-- Total Users to Process
	total := len(HornbillCache.UsersWorking)
	//-- Progress Bar
	bar := pb.StartNew(total)

	jobs := make(chan int, total)
	results := make(chan int, total)

	for w := 1; w <= Flags.configWorkers; w++ {
		go worker(w, jobs, results, bar)
	}

	for j := 1; j <= total; j++ {
		jobs <- j
	}
	close(jobs)
	//-- Finally we collect all the results of the work.
	for a := 1; a <= total; a++ {
		<-results
	}
	bar.FinishPrint("Finalising User Data Complete")
}

func createUser(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userCreate(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(1)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Create User: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}

}

func updateUser(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userUpdate(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(2)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Update User: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}
}

func updateUserProfile(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userProfileUpdate(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(3)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Update User Profile: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}
}

func updateUserImage(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userImageUpdate(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(4)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Update User Image: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}
}

func removeUserGroups(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userGroupsRemove(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(8)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Remove User Groups: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}
}
func updateUserGroups(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userGroupsUpdate(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(5)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Update User Groups: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}
}

func updateUserRoles(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userRolesUpdate(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(6)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Update User Roles: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}
}
func updateUserStatus(espXmlmc *apiLib.XmlmcInstStruct, currentUser *userWorkingDataStruct, buffer *bytes.Buffer) {
	b, err := userStatusUpdate(espXmlmc, currentUser, buffer)
	if b {
		CounterInc(9)
	} else {
		CounterInc(7)
		buffer.WriteString(loggerGen(4, "Unable to Update User Status: "+currentUser.Account.UserID+" Error: "+fmt.Sprintf("%s", err)))
	}
}
