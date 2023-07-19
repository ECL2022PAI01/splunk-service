package handler

import (
	"strings"
	"testing"

	"github.com/ECL2022PAI01/splunk-service/pkg/utils"
	keptnv1 "github.com/keptn/go-utils/pkg/lib"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"
)

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
	splunkServer := utils.BuildMockSplunkServer(defaultSplunkTestResult)
	defer splunkServer.Close()

	//setting splunk credentials
	env := utils.EnvConfig{}
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

	err = HandleConfigureMonitoringTriggeredEvent(ddKeptn, *incomingEvent, data, env)

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
