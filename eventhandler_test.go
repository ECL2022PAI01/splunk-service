package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event/datacodec"
	"github.com/joho/godotenv"
	keptnv1 "github.com/keptn/go-utils/pkg/lib"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	keptn "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	splunk "github.com/kuro-jojo/splunk-sdk-go/src/client"
	splunktest "github.com/kuro-jojo/splunk-sdk-go/tests"
)

// You can configure your tests by specifying the path to get-sli triggered event file in json,
// the path to your sli.yaml file
// and the default result you await from splunk
// Indicators given in get-sli.triggered.json should match indicators in the given sli file
const (
	getSliTriggeredEventFile              = "test/events/get-sli.triggered.json"
	configureMonitoringTriggeredEventFile = "test/events/monitoring.configure.json"
	sliFilePath                           = "./test/data/podtatohead.sli.yaml"
	defaultSplunkTestResult               = 1250
)

/**
 * loads a cloud event from the passed test json file and initializes a keptn object with it
 */
func initializeTestObjects(eventFileName string, resourceServiceUrl string) (*keptnv2.Keptn, *cloudevents.Event, error) {
	// load sample event
	eventFile, err := os.ReadFile(eventFileName)
	if err != nil {
		return nil, nil, fmt.Errorf("Can't load %s: %w", eventFileName, err)
	}

	incomingEvent := &cloudevents.Event{}
	err = json.Unmarshal(eventFile, incomingEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing: %w", err)
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
func TestHandleConfigureMonitoringTriggeredEvent(t *testing.T) {

	//Building a mock resource service server
	resourceServiceServer, err := buildMockResourceServiceServer(sliFilePath)
	if err != nil {
		t.Errorf("Error reading sli file : %s", err.Error())
		t.Fail()
	}
	defer resourceServiceServer.Close()

	//Building a mock splunk server
	splunkServer := buildMockSplunkServer()
	defer splunkServer.Close()

	//setting splunk credentials
	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"

	//Initializing test objects
	ddKeptn, incomingEvent, err := initializeTestObjects(configureMonitoringTriggeredEventFile, resourceServiceServer.URL+"/api/resource-service")
	if err != nil {
		t.Fatal(err)
	}

	if incomingEvent.Type() == keptnv1.ConfigureMonitoringEventType {
		incomingEvent.SetType(keptnv2.GetTriggeredEventType(keptnv2.ConfigureMonitoringTaskName))
	}

	data := &keptnv2.ConfigureMonitoringTriggeredEventData{}
	err = incomingEvent.DataAs(data)

	if err != nil {
		t.Fatal("Error getting keptn event data")
	}

	err = HandleConfigureMonitoringTriggeredEvent(ddKeptn, *incomingEvent, data)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	gotEvents := len(ddKeptn.EventSender.(*fake.EventSender).SentEvents)

	// Verify that HandleGetSliTriggeredEvent has sent 2 cloudevents
	if gotEvents != 2 {
		t.Fatalf("Expected two events to be sent, but got %v", gotEvents)
	}

	// Verify that the first CE sent is a .started event
	if keptnv2.GetStartedEventType(keptnv2.ConfigureMonitoringTaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type() {
		t.Fatal("Expected a configure-monitoring.started event type")
	}

	// Verify that the second CE sent is a .finished event
	if keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[1].Type() {
		t.Fatal("Expected a configure-monitoring.finished event type")
	}
}

// Tests the HandleSpecificSli function
func TestHandleSpecificSli(t *testing.T) {
	indicatorName := "test"
	data := &keptnv2.GetSLITriggeredEventData{}
	sliConfig := make(map[string]string, 1)
	sliConfig[indicatorName] = "test"

	//Building a mock splunk server returning default responses when getting  get and post requests

	splunkServer := buildMockSplunkServer()
	defer splunkServer.Close()

	//Retrieving the mock splunk server credentials
	splunkCreds := &splunkCredentials{
		Host:  strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1],
		Port:  strings.Split(splunkServer.URL, ":")[2],
		Token: "apiToken",
	}

	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		splunkCreds.Host,
		splunkCreds.Port,
		splunkCreds.Token,
		true,
	)
	sliResult, errored := handleSpecificSLI(client, indicatorName, data, sliConfig)

	if errored != nil {
		t.Fatal(errored.Error())
	}
	t.Logf("SLI Result : %v", sliResult.Value)
	if sliResult.Value != float64(defaultSplunkTestResult) {
		t.Fatalf("Wrong value for the metric %s : expected %v, got %v", indicatorName, defaultSplunkTestResult, sliResult.Value)
	}
}

// Tests the handleGetSliTriggered function
func TestHandleGetSliTriggered(t *testing.T) {

	//Building a mock resource service server
	resourceServiceServer, err := buildMockResourceServiceServer(sliFilePath)
	if err != nil {
		t.Fatalf("Error reading sli file : %s", err.Error())
	}
	defer resourceServiceServer.Close()

	//Building a mock splunk server
	splunkServer := buildMockSplunkServer()
	defer splunkServer.Close()

	//setting splunk credentials
	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"

	//Initializing test objects
	ddKeptn, incomingEvent, err := initializeTestObjects(getSliTriggeredEventFile, resourceServiceServer.URL+"/api/resource-service")
	if err != nil {
		t.Fatal(err)
	}

	data := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(data)

	if err != nil {
		t.Fatalf("Error while getting keptn event data : %s", err.Error())
	}

	err = HandleGetSliTriggeredEvent(ddKeptn, *incomingEvent, data)

	if err != nil {
		t.Fatalf("Error : %s", err.Error())
	}

	gotEvents := len(ddKeptn.EventSender.(*fake.EventSender).SentEvents)

	// Verify that HandleGetSliTriggeredEvent has sent 2 cloudevents
	if gotEvents != 2 {
		t.Fatalf("Expected two events to be sent, but got %v", gotEvents)
	}

	// Verify that the first CE sent is a .started event
	if keptnv2.GetStartedEventType(keptnv2.GetSLITaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type() {
		t.Fatal("Expected a get-sli.started event type")
	}

	// Verify that the second CE sent is a .finished event
	if keptnv2.GetFinishedEventType(keptnv2.GetSLITaskName) != ddKeptn.EventSender.(*fake.EventSender).SentEvents[1].Type() {
		t.Fatal("Expected a get-sli.finished event type")
	}

	// Verify thet the .finished event contains the sli results
	finishedEvent := ddKeptn.EventSender.(*fake.EventSender).SentEvents[1]
	var respData keptnv2.GetSLIFinishedEventData
	err = datacodec.Decode(context.Background(), finishedEvent.DataMediaType(), finishedEvent.Data(), &respData)
	if err != nil {
		t.Fatalf("Unable to decode data from the event : %v", err.Error())
	}
	// print respData
	if respData.GetSLI.IndicatorValues == nil {
		t.Fatal("No results added into the response event for the indicators.")
	} else {
		//printing SLI results if no error has occured
		for _, sliResult := range respData.GetSLI.IndicatorValues {
			if sliResult.Value != float64(defaultSplunkTestResult) {
				t.Fatalf("Wrong value for the metric %s : %v", sliResult.Metric, sliResult.Value)

			} else {
				t.Logf("SLI Results for indicator %s : %v", sliResult.Metric, sliResult.Value)
			}
		}
	}
}

// Tests the getSplunkCredentials function
func TestGetSplunkCredentials(t *testing.T) {

	godotenv.Load(".env.local")
	env.SplunkApiToken = os.Getenv("SPLUNK_API_TOKEN")
	env.SplunkHost = os.Getenv("SPLUNK_HOST")
	env.SplunkPort = os.Getenv("SPLUNK_PORT")

	sp, err := getSplunkCredentials()

	if err == nil {
		t.Logf("Splunk credentials : %v", sp)
		if sp.Host == "" || sp.Port == "" || sp.Token == "" {
			t.Fatal("If Host, Port or token are empty. An error should be returned")
		}
	} else {
		t.Logf("Received expected error : %s", err.Error())
	}
}

// Build a mock splunk server returning default responses when getting  get and post requests
func buildMockSplunkServer() *httptest.Server {

	jsonResponsePOST := `{
		"sid": "10"
	}`
	jsonResponseGET := `{
		"results":[{"theRequest":"` + fmt.Sprint(defaultSplunkTestResult) + `"}]
	}`
	splunkResponses := make([]map[string]interface{}, 2)
	splunkResponses[0] = map[string]interface{}{
		"POST": jsonResponsePOST,
	}
	splunkResponses[1] = map[string]interface{}{
		"GET": jsonResponseGET,
	}
	splunkServer := splunktest.MultitpleMockRequest(splunkResponses, true)

	return splunkServer
}

// Build a mock resource service server returning a response with the content of the sli file
func buildMockResourceServiceServer(filePath string) (*httptest.Server, error) {

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	jsonResourceFileResp := `{
		"resourceContent": "` + base64.StdEncoding.EncodeToString(fileContent) + `",
		"resourceURI": "sli.yaml",
		"metadata": {
		  "upstreamURL": "https://github.com/user/keptn.git",
		  "version": "1.0.0"
		}
	  }`

	resourceServiceServer := splunktest.MockRequest(jsonResourceFileResp, false)

	return resourceServiceServer, nil
}

