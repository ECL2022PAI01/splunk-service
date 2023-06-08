package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	api "github.com/keptn/go-utils/pkg/api/utils"
	keptn "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	splunktest "github.com/kuro-jojo/splunk-sdk-go/tests"
)

// You can configure it by adding your splunk credentials(host, port and token),
// the parameters of your resource-service(baseUrl, authToken,authHeader and scheme)
// and your custom indicators(table of strings)
const (
	//Splunk Configuration
	customSplunkApiToken = "eyJraWQiOiJzcGx1bmsuc2VjcmV0IiwiYWxnIjoiSFM1MTIiLCJ2ZXIiOiJ2MiIsInR0eXAiOiJzdGF0aWMifQ.eyJpc3MiOiJhZG1pbiBmcm9tIE5DRUwxNDExOTIiLCJzdWIiOiJhZG1pbiIsImF1ZCI6ImtlcHRuIiwiaWRwIjoiU3BsdW5rIiwianRpIjoiODBkOGFkNDQ4MWY3NWQwOTYzMjY3ZWM3NjAzNjQ1NDg4NDI0ZWE1YTkyZDk0NTYzNGRkNTk1NzU1YTk3YzEyZCIsImlhdCI6MTY4NTYwNTM2MywiZXhwIjoxNjg4MTk3MzYzLCJuYnIiOjE2ODU2MDUzNjN9.eLqkWeU6TQzmfMwoJY3E0USL36pxzUri7mst-HrQb2Ay3UgZpCBfUdEM6BZ-Qgfm1gLxvGWKBsqDPGezBeiuhg"
	customSplunkHost     = "172.29.226.241"
	customSplunkPort     = "8089"
	//Resource-service configuration
	customResourceServiceBaseUrl    = "http://localhost:8090/api/resource-service"
	customResourceServiceScheme     = "http"
	customResourceServiceAuthToken  = "nBsd0T3fHwX8csWJQPgwAXlTJBJzL2z4xK1LAgnBfvMdb"
	customResourceServiceAuthHeader = "x-token"
	//don't forget to specify the right parameters for your get-sli.triggered event in test/events/get-sli.triggered.json or specify the path to your event.json file

)

var (
	//Indicators
	customIndicators = []string{"number_of_logs", "number_of_logs2"}
)

/**
 * loads a cloud event from the passed test json file and initializes a keptn object with it
 */
func initializeTestObjects(eventFileName string) (*keptnv2.Keptn, *cloudevents.Event, error) {
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
	keptnOptions.UseLocalFileSystem = true
	keptnOptions.ConfigurationServiceURL = customResourceServiceBaseUrl

	ddKeptn, err := keptnv2.NewKeptn(incomingEvent, keptnOptions)
	if err == nil {
		ddKeptn.ResourceHandler = &api.ResourceHandler{
			BaseURL:    customResourceServiceBaseUrl,
			HTTPClient: nil,
			Scheme:     customResourceServiceScheme,
		}
		ddKeptn.ResourceHandler = api.NewAuthenticatedResourceHandler(customResourceServiceBaseUrl, customResourceServiceAuthToken, customResourceServiceAuthHeader, nil, customResourceServiceScheme)
	}

	return ddKeptn, incomingEvent, err
}

// Tests the HandleGetSliTriggeredEvent Handler
// TODO: Add your test-code
func TestHandleGetSliTriggered(t *testing.T) {

	ddKeptn, incomingEvent, err := initializeTestObjects("test/events/get-sli.triggered.json")
	if err != nil {
		t.Error(err)
		return
	}

	data := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(data)
	data.GetSLI.Indicators = customIndicators

	if err != nil {
		t.Errorf("Error getting keptn event data")
		t.Fail()
	}

	env.SplunkHost = customSplunkHost
	env.SplunkApiToken = customSplunkApiToken
	env.SplunkPort = customSplunkPort

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
}

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
	jsonResponsePOST := `{
		"sid": "10"
	}`
	jsonResponseGET := `{
		"results":[{"count":"` + fmt.Sprint(1250) + `"}]
	}`

	responses := make([]map[string]interface{}, 2)
	responses[0] = map[string]interface{}{
		"POST": jsonResponsePOST,
	}
	responses[1] = map[string]interface{}{
		"GET": jsonResponseGET,
	}
	server := splunktest.MutitpleMockRequest(responses)
	defer server.Close()

	//Retrieving the mock splunk server credentials
	splunkCreds := &splunkCredentials{
		Host:  strings.Split(strings.Split(server.URL, ":")[1], "//")[1],
		Port:  strings.Split(server.URL, ":")[2],
		Token: "apiToken",
	}

	wg.Add(1)
	go handleSpecificSLI(indicatorName, splunkCreds, data, sliConfig, &sliResults, &errored)
	wg.Wait()

	if len(sliResults) == 0 || sliResults[0].Value != splunkResult {
		t.Error("Expected to add a keptnv2.SLIResult to sliResults but nothing added.")
	}

}
