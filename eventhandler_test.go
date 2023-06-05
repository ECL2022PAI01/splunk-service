package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/kuro-jojo/go-utils/pkg/lib/v0_2_0/fake"

	keptn "github.com/kuro-jojo/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/kuro-jojo/go-utils/pkg/lib/v0_2_0"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
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
	keptnOptions.ConfigurationServiceURL = "http://localhost:8010"

	ddKeptn, err := keptnv2.NewKeptn(incomingEvent, keptnOptions)

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

	specificEvent := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(specificEvent)
	if err != nil {
		t.Errorf("Error getting keptn event data")
		t.Fail()
	}

	env.SplunkApiToken = "eyJraWQiOiJzcGx1bmsuc2VjcmV0IiwiYWxnIjoiSFM1MTIiLCJ2ZXIiOiJ2MiIsInR0eXAiOiJzdGF0aWMifQ.eyJpc3MiOiJhZG1pbiBmcm9tIE5DRUwxNDExOTIiLCJzdWIiOiJhZG1pbiIsImF1ZCI6ImtlcHRuIiwiaWRwIjoiU3BsdW5rIiwianRpIjoiODBkOGFkNDQ4MWY3NWQwOTYzMjY3ZWM3NjAzNjQ1NDg4NDI0ZWE1YTkyZDk0NTYzNGRkNTk1NzU1YTk3YzEyZCIsImlhdCI6MTY4NTYwNTM2MywiZXhwIjoxNjg4MTk3MzYzLCJuYnIiOjE2ODU2MDUzNjN9.eLqkWeU6TQzmfMwoJY3E0USL36pxzUri7mst-HrQb2Ay3UgZpCBfUdEM6BZ-Qgfm1gLxvGWKBsqDPGezBeiuhg"
	env.SplunkHost = "172.29.226.241"
	env.SplunkPort = "8089"

	err = HandleGetSliTriggeredEvent(ddKeptn, *incomingEvent, specificEvent)
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
