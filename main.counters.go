package main

func errorCountInc() {
	mutexCounters.Lock()
	errorCount++
	mutexCounters.Unlock()
}
func updateCountInc() {
	mutexCounters.Lock()
	counters.updated++
	mutexCounters.Unlock()
}
func updateSkippedCountInc() {
	mutexCounters.Lock()
	counters.updatedSkipped++
	mutexCounters.Unlock()
}
func createSkippedCountInc() {
	mutexCounters.Lock()
	counters.createskipped++
	mutexCounters.Unlock()
}
func createCountInc() {
	mutexCounters.Lock()
	counters.created++
	mutexCounters.Unlock()
}
func profileCountInc() {
	mutexCounters.Lock()
	counters.profileUpdated++
	mutexCounters.Unlock()
}
func profileSkippedCountInc() {
	mutexCounters.Lock()
	counters.profileSkipped++
	mutexCounters.Unlock()
}
