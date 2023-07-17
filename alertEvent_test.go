package main

import (
	"strings"
	"testing"
)

func TestFiringAlertsPoll(t *testing.T) {
	//Building a mock splunk server
	splunkServer := builMockSplunkServer()
	defer splunkServer.Close()

	//setting splunk credentials
	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"
	
	go FiringAlertsPoll()

	

}