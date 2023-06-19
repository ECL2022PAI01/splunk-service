package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	keptnv1 "github.com/keptn/go-utils/pkg/lib"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)


var calledSLI bool
var calledConfig bool
//Tests the getSplunkCredentials function
func TestProcessKeptnCloudEvent(t *testing.T){

	t.Log("Initializing get sli triggered event")

	handleConfMonitEvent = func(ddKeptn *keptnv2.Keptn, incomingEvent event.Event, data *keptnv2.ConfigureMonitoringTriggeredEventData) error{
		calledConfig = true
		return nil
	}
	handleGetSli = func(ddKeptn *keptnv2.Keptn, incomingEvent event.Event, data *keptnv2.GetSLITriggeredEventData) error{
		calledSLI = true
		return nil
	}

	calledSLI = false
	calledConfig = false
	incomingEvent, _:= extractEvent("test/events/get-sli.triggered.json")
	checkProcessKeptnCloudEvent(t, incomingEvent)

	calledSLI = false
	calledConfig = false
	incomingEvent, _= extractEvent("test/events/monitoring.configure.json")
	checkProcessKeptnCloudEvent(t, incomingEvent)

	calledSLI = false
	calledConfig = false
	incomingEvent, _= extractEvent("test/events/release.triggered.json")
	checkProcessKeptnCloudEvent(t, incomingEvent)

}

func extractEvent(eventFileName string) (*event.Event, error){

	//eventFileName:= "test/events/get-sli.triggered.json"
	eventFile, err := ioutil.ReadFile(eventFileName)
	if err != nil {
		fmt.Printf("Cant load %s: %s", eventFileName, err.Error())
		return nil, err
	}

	incomingEvent := &cloudevents.Event{}
	err = json.Unmarshal(eventFile, incomingEvent)
	if err != nil {
		fmt.Printf("Error parsing: %s", err.Error())
		return nil, err
	}

	return incomingEvent, err

}

func checkProcessKeptnCloudEvent(t *testing.T, incomingEvent *event.Event){
	
	err:= processKeptnCloudEvent(context.Background(), *incomingEvent)
	
	if err==nil{
		if(incomingEvent.Type()==keptnv2.ConfigureMonitoringTaskName){
			if(calledConfig==false){
				t.Errorf("The configure monitoring event has not been handled.")
				t.Fail()
			}
		}else if(incomingEvent.Type()==keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName)){
			if(calledSLI==false){
				t.Errorf("The get-sli triggered event has not been handled.")
				t.Fail()
			}
		} else if(incomingEvent.Type()==keptnv1.ConfigureMonitoringEventType){
			t.Errorf("keptnv1 configure monitoring event must be converted into keptnv2 configure monitoring event.")
			t.Fail()
		}
	}else{
		if strings.HasPrefix(err.Error(), "Unhandled Keptn Cloud Event:") &&
		(incomingEvent.Type()==keptnv2.ConfigureMonitoringTaskName ||
		incomingEvent.Type()==keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName) ||
		incomingEvent.Type()==keptnv1.ConfigureMonitoringEventType){
			t.Errorf("The function didn't handle an event that should have been handled.")
			t.Fail()
		}
	}

}