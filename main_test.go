package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/kelseyhightower/envconfig"
	keptnv1 "github.com/keptn/go-utils/pkg/lib"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	logger "github.com/sirupsen/logrus"
)


var calledSLI bool
var calledConfig bool
var testPortforMain = 8090

// Tests the parseKeptnCloudEventPayload function
func TestParseKeptnCloudEventPayload(t *testing.T){
	incomingEvent, err:= extractEvent("test/events/get-sli.triggered.json")
	if err!= nil{
		t.Errorf("Error getting keptn event : %s", err.Error())
	}
	eventData := &keptnv2.GetSLITriggeredEventData{}
	err = parseKeptnCloudEventPayload(*incomingEvent, eventData)

	//fails if eventData has not been modified
	if(err!=nil || eventData.Project==""){
		t.Errorf("Failed to parse keptn cloud event payload")
		t.Fail()
	}
	t.Logf("%v", eventData)
}

// Tests the processKeptnCloudEvent function by checking if it processes get-sli.triggered and configure monitoring events
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

	//Test for a get-sli.triggered event
	calledSLI = false
	calledConfig = false
	checkProcessKeptnCloudEvent(t, "test/events/get-sli.triggered.json")

	//Test for a monitoring.configure event
	calledSLI = false
	calledConfig = false
	checkProcessKeptnCloudEvent(t, "test/events/monitoring.configure.json")

	//Test for a random event
	calledSLI = false
	calledConfig = false
	checkProcessKeptnCloudEvent(t, "test/events/release.triggered.json")

}

//Tests the _main function by ensuring that it listens to cloudevents and trigger the procKeptnCE function
func Test_main(t *testing.T){

	if err := envconfig.Process("", &env); err != nil {
		logger.Fatalf("Failed to process env var: %s", err)
	}
	env.Port = testPortforMain

	handled := false
	procKeptnCE = func(ctx context.Context, event cloudevents.Event) error {
		handled= true
		return nil
	}
	args := []string{}
	go _main(args)
	//sleep for 2 seconds to let the previous go routine the time to start listening for events
	time.Sleep(time.Duration(2) * time.Second)
	err := sendTestCloudEvent("test/events/get-sli.triggered.json")
	if err!=nil{
		logger.Fatalf("Couldn't send cloud event : %s", err.Error())
	}

	if(handled==false){
		t.Error("The function didn't handle the event.")
		t.Fail()
	}

}

//Tests the main function by verifying if the exit code corresponds to the one returned by call_main function
func TestMain(t *testing.T) {
	const expectedReturn = 15

	call_main = func(args []string) int {
		return expectedReturn
	}
    if os.Getenv("BE_MAIN") == "1" {
        main()
        return
    }
	
    cmd := exec.Command(os.Args[0], "-test.run=TestMain")
    cmd.Env = append(os.Environ(), "BE_MAIN=1")
    err := cmd.Run()
    if e, ok := err.(*exec.ExitError); ok && e.ExitCode()==expectedReturn {
        return
    }
    t.Fatalf("process ran with err %v, want exit status "+fmt.Sprint(expectedReturn), err)
}

//reads the json event file and convert its content into an event
func extractEvent(eventFileName string) (*event.Event, error){

	eventFile, err := ioutil.ReadFile(eventFileName)
	if err != nil {
		return nil, err
	}

	incomingEvent := &cloudevents.Event{}
	err = json.Unmarshal(eventFile, incomingEvent)
	if err != nil {
		return nil, err
	}

	return incomingEvent, err

}

//Check if events are handled (not handled) when they should be (shouldn't be)
func checkProcessKeptnCloudEvent(t *testing.T, fileName string){
	
	incomingEvent, err:= extractEvent(fileName)
	if err!= nil{
		t.Errorf("Error getting keptn event : %s", err.Error())
	}

	err= processKeptnCloudEvent(context.Background(), *incomingEvent)
	
	if err==nil{
		//verify if events that should be handled are handled correctly
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
		//verify if events that should be handled are not skipped
		if strings.HasPrefix(err.Error(), UnhanKeptnCE) &&
		(incomingEvent.Type()==keptnv2.ConfigureMonitoringTaskName ||
		incomingEvent.Type()==keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName) ||
		incomingEvent.Type()==keptnv1.ConfigureMonitoringEventType){
			t.Errorf("The function didn't handle an event that should have been handled.")
			t.Fail()
		}
	}

}

//Sends a cloud event
func sendTestCloudEvent(eventFileName string) error{
	body, err := ioutil.ReadFile(eventFileName)
	if err != nil {
		fmt.Printf("Cant load %s: %s", eventFileName, err.Error())
		return err
	}
    client := &http.Client{}
    req, err := http.NewRequest("POST", "http://localhost:"+fmt.Sprint(testPortforMain), bytes.NewBuffer(body))
    if err != nil {
        fmt.Printf("Error : %s\n", err)
        return err
    }
    req.Header.Add("Accept", "application/json")
    req.Header.Add("Cache-Control", "no-cache")
    req.Header.Add("Content-Type", "application/cloudevents+json")
    res, err := client.Do(req)
    if err != nil {
        return err
    }

    defer res.Body.Close()
    bo, err := io.ReadAll(res.Body)
    if err != nil {
        return err
    }
    fmt.Printf("Response : %s", bo)
	return nil
}