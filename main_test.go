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
var testPortforMain = 38888

func TestParseKeptnCloudEventPayload(t *testing.T){
	incomingEvent, err:= extractEvent("test/events/get-sli.triggered.json")
	if err!= nil{
		t.Errorf("Error getting keptn event : %s", err.Error())
	}
	eventData := &keptnv2.GetSLITriggeredEventData{}
	err = parseKeptnCloudEventPayload(*incomingEvent, eventData)

	if(err!=nil || eventData.Project==""){
		t.Errorf("Failed to parse keptn cloud event payload")
		t.Fail()
	}
	t.Logf("%v", eventData)
}

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
	incomingEvent, err:= extractEvent("test/events/get-sli.triggered.json")
	if err!= nil{
		t.Errorf("Error getting keptn event : %s", err.Error())
	}
	checkProcessKeptnCloudEvent(t, incomingEvent)

	calledSLI = false
	calledConfig = false
	incomingEvent, err= extractEvent("test/events/monitoring.configure.json")
	if err!= nil{
		t.Errorf("Error getting keptn event : %s", err.Error())
	}
	checkProcessKeptnCloudEvent(t, incomingEvent)

	calledSLI = false
	calledConfig = false
	incomingEvent, err= extractEvent("test/events/release.triggered.json")
	if err!= nil{
		t.Errorf("Error getting keptn event : %s", err.Error())
	}
	checkProcessKeptnCloudEvent(t, incomingEvent)

}

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

func extractEvent(eventFileName string) (*event.Event, error){

	//eventFileName:= "test/events/get-sli.triggered.json"
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
		if strings.HasPrefix(err.Error(), UnhanKeptnCE) &&
		(incomingEvent.Type()==keptnv2.ConfigureMonitoringTaskName ||
		incomingEvent.Type()==keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName) ||
		incomingEvent.Type()==keptnv1.ConfigureMonitoringEventType){
			t.Errorf("The function didn't handle an event that should have been handled.")
			t.Fail()
		}
	}

}

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
