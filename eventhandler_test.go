package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"

	keptn "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

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
}

func TestHandleSpecificSli(t *testing.T){
	indicatorName := "test"
	splunkCreds := &splunkCredentials{Host : "localhost", Port: "8089", Token: "random"}
	data := &keptnv2.GetSLITriggeredEventData{}
	sliResults := []*keptnv2.SLIResult{}
	errored := false
	sliConfig := make(map[string]string, 1)
	sliConfig["test"] = "test"

	wg.Add(1)
	go handleSpecificSLI(indicatorName, splunkCreds, data, sliConfig, &sliResults, &errored)
	wg.Wait()

	if len(sliResults)!=1{
		t.Error("Ëxpected to add a keptnv2.SLIResult to sliResults but nothing added.")
	}

	if len(sliResults)==1 && reflect.TypeOf(sliResults[0]).String()!= "*github.com/keptn/go-utils/pkg/lib/v0_2_0.SLIResult"{
		t.Errorf("Ëxpected to add a keptnv2.SLIResult to sliResults but %T was added, %v.", sliResults[0], reflect.TypeOf(sliResults[0]).String())
	}

}
