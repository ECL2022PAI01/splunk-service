package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event/datacodec"
	"github.com/joho/godotenv"
	keptnv1 "github.com/keptn/go-utils/pkg/lib"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	keptn "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	splunk "github.com/kuro-jojo/splunk-sdk-go/client"
	splunktest "github.com/kuro-jojo/splunk-sdk-go/tests"
)

// You can configure your tests by specifying the path to get-sli triggered event file in json,
// the path to your sli.yaml file
// and the default result you await from splunk
// Indicators given in get-sli.triggered.json should match indicators in the given sli file
const (
	GET_SLI_TRIGGERED_EVENT = "test/events/get-sli.triggered.json"
	CONF_MONITORING_TRIGG_EVENT = "test/events/monitoring.configure.json"
	SLI_FILE_PATH              = "./test/data/sli.yaml"
	DEFAULT_SPLUNK_RESULT  = 1250
)

/**
 * loads a cloud event from the passed test json file and initializes a keptn object with it
 */
func initializeTestObjects(eventFileName string, resourceServiceUrl string) (*keptnv2.Keptn, *cloudevents.Event, error) {
	// load sample event
	eventFile, err := ioutil.ReadFile(eventFileName)
	if err != nil {
		return nil, nil, fmt.Errorf("Cant load %s: %s", eventFileName, err.Error())
	}

	incomingEvent := &cloudevents.Event{}
	err = json.Unmarshal(eventFile, incomingEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing: %s", err.Error())
	}

	// Add a Fake EventSender to KeptnOptions
	var keptnOptions = keptn.KeptnOpts{
		EventSender: &fake.EventSender{},
	}
	keptnOptions.ConfigurationServiceURL = resourceServiceUrl
	keptnOptions.UseLocalFileSystem = true

	ddKeptn, err := keptnv2.NewKeptn(incomingEvent, keptnOptions)

	return ddKeptn, incomingEvent, err
}

// Tests the HandleMonitoringTriggeredEvent
func TestHandleConfigureMonitoringTriggeredEvent(t *testing.T){

	ddKeptn, incomingEvent, err := initializeTestObjects(CONF_MONITORING_TRIGG_EVENT, "")
	if err != nil {
		t.Error(err)
		return
	}

	if incomingEvent.Type() == keptnv1.ConfigureMonitoringEventType {
		incomingEvent.SetType(keptnv2.GetTriggeredEventType(keptnv2.ConfigureMonitoringTaskName))
	}

	data := &keptnv2.ConfigureMonitoringTriggeredEventData{}
	err = incomingEvent.DataAs(data)


	if err != nil {
		t.Errorf("Error getting keptn event data")
		t.Fail()
	}

	err = HandleConfigureMonitoringTriggeredEvent(ddKeptn, *incomingEvent, data)

	if err != nil {
		t.Errorf("Error: " + err.Error())
		t.Fail()
	}

	gotEvents := len(ddKeptn.EventSender.(*fake.EventSender).SentEvents)

	// Verify that HandleGetSliTriggeredEvent has sent 2 cloudevents
	if gotEvents != 2 {
		t.Errorf("Expected two events to be sent, but got %v", gotEvents)
		t.Fail()
	}

	// Verify that the first CE sent is a .started event
	if keptnv2.GetStartedEventType(keptnv2.ConfigureMonitoringTaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type() {
		t.Errorf("Expected a configureMonitoring.started event type")
		t.Fail()
	}

	// Verify that the second CE sent is a .finished event
	if keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[1].Type() {
		t.Errorf("Expected a configureMonitoring.finished event type")
		t.Fail()
	}


}

// Tests the HandleGetSliTriggeredEvent Handler
func TestHandleGetSliTriggered(t *testing.T) {

	//Building a mock resource service server
	resourceServiceServer, err := buildMockResourceServiceServer(SLI_FILE_PATH)
	if err != nil {
		t.Errorf("Error reading sli file : %s", err.Error())
		t.Fail()
	}
	defer resourceServiceServer.Close()

	//Building a mock splunk server
	splunkServer := builMockSplunkServer()
	defer splunkServer.Close()

	//setting splunk credentials
	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"

	//Initializing test objects
	t.Logf("INFO : %s", splunkServer.URL)
	t.Logf("INFO : %s", resourceServiceServer.URL)
	//time.Sleep(2 * time.Minute)
	ddKeptn, incomingEvent, err := initializeTestObjects(GET_SLI_TRIGGERED_EVENT, resourceServiceServer.URL+"/api/resource-service")
	if err != nil {
		t.Error(err)
		return
	}

	data := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(data)

	if err != nil {
		t.Errorf("Error getting keptn event data")
		t.Fail()
	}

	err = HandleGetSliTriggeredEvent(ddKeptn, *incomingEvent, data)

	if err != nil {
		t.Errorf("Error: " + err.Error())
		t.Fail()
	}

	gotEvents := len(ddKeptn.EventSender.(*fake.EventSender).SentEvents)

	// Verify that HandleGetSliTriggeredEvent has sent 2 cloudevents
	if gotEvents != 2 {
		t.Errorf("Expected two events to be sent, but got %v", gotEvents)
		t.Fail()
	}

	// Verify that the first CE sent is a .started event
	if keptnv2.GetStartedEventType(keptnv2.GetSLITaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type() {
		t.Errorf("Expected a get-sli.started event type")
		t.Fail()
	}

	// Verify that the second CE sent is a .finished event
	if keptnv2.GetFinishedEventType(keptnv2.GetSLITaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[1].Type() {
		t.Errorf("Expected a get-sli.finished event type")
		t.Fail()
	}

	// Verify thet the .finished event contains the sli results
	finishedEvent := ddKeptn.EventSender.(*fake.EventSender).SentEvents[1]
	var respData keptnv2.GetSLIFinishedEventData
	err = datacodec.Decode(context.Background(), finishedEvent.DataMediaType(), finishedEvent.Data(), &respData)
	if err != nil {
		t.Errorf("Unable to decode data from the event : %v", err.Error())
		t.Fail()
	}
	if respData.GetSLI.IndicatorValues == nil {
		t.Errorf("No results added into the response event for the indicators.")
	}else{
		//printing SLI results if no error has occured
		for _, sliResult := range respData.GetSLI.IndicatorValues {
			if(sliResult.Value != float64(DEFAULT_SPLUNK_RESULT)){
				t.Errorf("Wrong value for the metric %s : %v", sliResult.Metric, sliResult.Value)
			}
		}
	}

	//printing SLI results if no error has occured
	for _, sliResult := range respData.GetSLI.IndicatorValues {
		t.Logf("SLI Results for indicator %s : %v", sliResult.Metric, sliResult.Value)
	}

}

// Tests the HandleSpecificSli function
func TestHandleSpecificSli(t *testing.T) {

	indicatorName := "test"
	data := &keptnv2.GetSLITriggeredEventData{}
	sliResults := []*keptnv2.SLIResult{}
	errored := false
	sliConfig := make(map[string]string, 1)
	sliConfig[indicatorName] = "test"

	//Building a mock splunk server returning default responses when getting  get and post requests

	splunkServer := builMockSplunkServer()
	defer splunkServer.Close()

	//Retrieving the mock splunk server credentials
	splunkCreds := &splunkCredentials{
		Host:  strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1],
		Port:  strings.Split(splunkServer.URL, ":")[2],
		Token: "apiToken",
	}

	wg.Add(1)
	go handleSpecificSLI(indicatorName, splunkCreds, data, sliConfig, &sliResults, &errored)
	wg.Wait()

	if len(sliResults) == 0 || sliResults[0].Value != DEFAULT_SPLUNK_RESULT {
		t.Error("Expected to add a keptnv2.SLIResult to sliResults but nothing added.")
	}

}

//Tests the retrieveSearchTimeRange function
func TestRetrieveSearchTimeRange(t *testing.T){

	const DEFAULT_EARLIEST_IN_QUERY = "-2m"
	const DEFAULT_LATEST_IN_QUERY = "+2m"
	const DEFAULT_EARLIEST_IN_PARAMS = "-1m"
	const DEFAULT_LATEST_IN_PARAMS = "+1m"

	splunkRequestParams:= &splunk.RequestParams{}

	//Verify if the function overwrites the time values in params and set theme to the values specified in the search query
	searchQuery:= "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure earliest="+DEFAULT_EARLIEST_IN_QUERY+" latest="+DEFAULT_LATEST_IN_QUERY+" |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, searchQuery, DEFAULT_EARLIEST_IN_QUERY, DEFAULT_LATEST_IN_QUERY, DEFAULT_EARLIEST_IN_PARAMS, DEFAULT_LATEST_IN_PARAMS)
	
	//Verify if the function overwrites only the latest time value in params
	searchQuery= "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure latest="+DEFAULT_LATEST_IN_QUERY+" |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, searchQuery, DEFAULT_EARLIEST_IN_PARAMS, DEFAULT_LATEST_IN_QUERY, DEFAULT_EARLIEST_IN_PARAMS, DEFAULT_LATEST_IN_PARAMS)

	//Verify if the function keeps the default values in params
	searchQuery= "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, searchQuery, DEFAULT_EARLIEST_IN_PARAMS, DEFAULT_LATEST_IN_PARAMS, DEFAULT_EARLIEST_IN_PARAMS, DEFAULT_LATEST_IN_PARAMS)
	
	//Verify if the function overwrites only the earliest time value in params
	searchQuery= "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure earliest="+DEFAULT_EARLIEST_IN_QUERY+" |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, searchQuery, DEFAULT_EARLIEST_IN_QUERY, DEFAULT_LATEST_IN_PARAMS, DEFAULT_EARLIEST_IN_PARAMS, DEFAULT_LATEST_IN_PARAMS)

	//Verify if the function ignores the second earliest time given in the query
	searchQuery= "source=/opt/splunk/var/log/secure.log sourcetype=osx_secure earliest="+DEFAULT_EARLIEST_IN_QUERY+" earliest="+DEFAULT_EARLIEST_IN_PARAMS+" |stats count"
	checkRetrieveSearchTimeRange(t, splunkRequestParams, searchQuery, DEFAULT_EARLIEST_IN_QUERY, DEFAULT_LATEST_IN_PARAMS, DEFAULT_EARLIEST_IN_PARAMS, DEFAULT_LATEST_IN_PARAMS)

}

//Tests the getSplunkCredentials function
func TestGetSplunkCredentials(t *testing.T){

	godotenv.Load(".env.local")
	env.SplunkApiToken = os.Getenv("SPLUNK_API_TOKEN")
	env.SplunkHost = os.Getenv("SPLUNK_HOST")
	env.SplunkPort = os.Getenv("SPLUNK_PORT")

	sp, err := getSplunkCredentials()

	if err== nil{
		t.Logf("Splunk credentials : %v", sp)
		if(sp.Host=="" || sp.Port=="" || sp.Token==""){
			t.Errorf("If Host, Port or token are empty. An error should be returned")
			t.Fail()
		}
	}else{
		t.Logf("Received expected error : %s", err.Error())
	}

}

// Build a mock splunk server returning default responses when getting  get and post requests
func builMockSplunkServer() *httptest.Server {

	jsonResponsePOST := `{
		"sid": "10"
	}`
	jsonResponseGET := `{
		"results":[{"theRequest":"` + fmt.Sprint(DEFAULT_SPLUNK_RESULT) + `"}]
	}`
	splunkResponses := make([]map[string]interface{}, 2)
	splunkResponses[0] = map[string]interface{}{
		"POST": jsonResponsePOST,
	}
	splunkResponses[1] = map[string]interface{}{
		"GET": jsonResponseGET,
	}
	splunkServer := splunktest.MutitpleMockRequest(splunkResponses, true)
	return splunkServer

}

// Build a mock resource service server returning a response with the content of the sli file
func buildMockResourceServiceServer(filePath string) (*httptest.Server, error) {

	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	jsonResourceFileResp := `{
		"resourceContent": "` + base64.StdEncoding.EncodeToString(fileContent) + `",
		"resourceURI": "sli.yaml",
		"metadata": {
		  "upstreamURL": "https://github.com/user/keptn.git",
		  "version": "somethingugly"
		}
	  }`

	resourceServiceServer := splunktest.MockRequest(jsonResourceFileResp, false)

	return resourceServiceServer, nil

}

//ckeck if we have the expected values in params after executing retrieveSearchTimeRange function
func checkRetrieveSearchTimeRange(t *testing.T, splunkRequestParams *splunk.RequestParams, newSearchQuery string, awaitedFinaleEarliest string, awaitedFinaleLatest string, defaultEarliest string, defaultLatest string){

	splunkRequestParams.SearchQuery= newSearchQuery
	splunkRequestParams.EarliestTime= defaultEarliest
	splunkRequestParams.LatestTime= defaultLatest
	retrieveSearchTimeRange(splunkRequestParams)

	if(splunkRequestParams.EarliestTime!=awaitedFinaleEarliest || splunkRequestParams.LatestTime!=awaitedFinaleLatest){
		t.Errorf("EarliestTime value %s and LatestTime value %s in params are incorrect, should be %s and %s.",
		splunkRequestParams.EarliestTime, splunkRequestParams.LatestTime, awaitedFinaleEarliest, awaitedFinaleLatest)
		t.Fail()
	}else{
		t.Log("Checked")
	}

}
