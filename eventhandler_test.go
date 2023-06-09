package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event/datacodec"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	keptn "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	splunktest "github.com/kuro-jojo/splunk-sdk-go/tests"
)

// You can configure your tests by specifying the path to get-sli triggered event file in json,
// and the path to your sli.yaml file
// Indicators given in get-sli.triggered.json should match indicators in the given sli file
const (
	getSliTriggeredEventFile= "test/events/get-sli.triggered.json"
	sliFilePath = "./test/data/sli.yaml"
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
	keptnOptions.ConfigurationServiceURL= resourceServiceUrl
	keptnOptions.UseLocalFileSystem = true

	ddKeptn, err := keptnv2.NewKeptn(incomingEvent, keptnOptions)

	return ddKeptn, incomingEvent, err
}

// Tests the HandleGetSliTriggeredEvent Handler
func TestHandleGetSliTriggered(t *testing.T) {
	
	//Building a mock resource service server
	resourceServiceServer, err:= buildMockResourceServiceServer(sliFilePath)
	if err!=nil{
		t.Errorf("Error reading sli file : %s",err.Error())
		t.Fail()
	}
	defer resourceServiceServer.Close()

	//Building a mock splunk server
	splunkServer:= builMockSplunkServer()
	defer splunkServer.Close()

	//setting splunk credentials
	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"

	//Initializing test objects
	t.Logf("INFO : %s",splunkServer.URL)
	t.Logf("INFO : %s",resourceServiceServer.URL)
	ddKeptn, incomingEvent, err := initializeTestObjects(getSliTriggeredEventFile, resourceServiceServer.URL)
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

	finishedEvent:= ddKeptn.EventSender.(*fake.EventSender).SentEvents[1]
	var respData keptnv2.GetSLIFinishedEventData
	err= datacodec.Decode(context.Background(), finishedEvent.DataMediaType(), finishedEvent.Data(), respData)
	//add another test
	if(err!=nil){
		t.Errorf("Unable to decode data from the event : %v", err.Error())
		t.Fail()
	}
	if(respData.GetSLI.IndicatorValues!=nil ){
		t.Errorf("Y a rien")
	}

}

//Tests the HandleSpecificSli function
func TestHandleSpecificSli(t *testing.T) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	indicatorName := "test"
	data := &keptnv2.GetSLITriggeredEventData{}
	sliResults := []*keptnv2.SLIResult{}
	errored := false
	sliConfig := make(map[string]string, 1)
	sliConfig[indicatorName] = "test"
	splunkResult := 1250.0

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

	if len(sliResults) == 0 || sliResults[0].Value != splunkResult {
		t.Error("Expected to add a keptnv2.SLIResult to sliResults but nothing added.")
	}

}

//Build a mock splunk server returning default responses when getting  get and post requests
func builMockSplunkServer()*httptest.Server{

	jsonResponsePOST := `{
		"sid": "10"
	}`
	jsonResponseGET := `{
		"results":[{"theRequest":"` + fmt.Sprint(1250) + `"}]
	}`
	splunkResponses := make([]map[string]interface{}, 2)
	splunkResponses[0] = map[string]interface{}{
		"POST": jsonResponsePOST,
	}
	splunkResponses[1] = map[string]interface{}{
		"GET": jsonResponseGET,
	}
	splunkServer := splunktest.MutitpleMockRequest(splunkResponses)
	return splunkServer

}

//Build a mock resource service server returning a response with the content of the sli file
func buildMockResourceServiceServer(filePath string) (*httptest.Server, error){

	fileContent, err:= ioutil.ReadFile(filePath)
	if err!=nil{
		return nil, err
	}
	jsonResourceFileResp := `{
		"resourceContent": "`+base64.StdEncoding.EncodeToString(fileContent)+`",
		"resourceURI": "sli.yaml",
		"metadata": {
		  "upstreamURL": "https://github.com/user/keptn.git",
		  "version": "somethingugly"
		}
	  }`
	
	resourceServiceServer := splunktest.MockRequest(jsonResourceFileResp)
	return resourceServiceServer, nil

}
