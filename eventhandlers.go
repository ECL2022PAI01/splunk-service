package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	splunk "github.com/kuro-jojo/splunk-sdk-go/client"
	splunkjob "github.com/kuro-jojo/splunk-sdk-go/jobs"
	logger "github.com/sirupsen/logrus"
)

const (
	SLI_FILE = "sli.yaml"
)

// Waitgroup structure needed to be able to use go routines in order to avoid waiting for a metric before executing the next one
var wg sync.WaitGroup
var mutex = &sync.RWMutex{}

type splunkCredentials struct {
	Host  string `json:"host" yaml:"spHost"`
	Token string `json:"token" yaml:"spToken"`
	Port  string `json:"port" yaml:"spPort"`
}

// HandleGetSliTriggeredEvent handles get-sli.triggered events if SLIProvider == splunk
func HandleGetSliTriggeredEvent(ddKeptn *keptnv2.Keptn, incomingEvent cloudevents.Event, data *keptnv2.GetSLITriggeredEventData) error {
	var shkeptncontext string
	_ = incomingEvent.Context.ExtensionAs("shkeptncontext", &shkeptncontext)
	configureLogger(incomingEvent.Context.GetID(), shkeptncontext)

	logger.Infof("Handling get-sli.triggered Event: %s", incomingEvent.Context.GetID())

	// Step 1 - Do we need to do something?
	// Lets make sure we are only processing an event that really belongs to our SLI Provider
	if data.GetSLI.SLIProvider != "splunk" {
		logger.Infof("Not handling get-sli event as it is meant for %s", data.GetSLI.SLIProvider)
		return nil
	}

	// Step 2 - Send out a get-sli.started CloudEvent
	// The get-sli.started cloud-event is new since Keptn 0.8.0 and is required to be send when the task is started
	_, err := ddKeptn.SendTaskStartedEvent(data, ServiceName)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send task started CloudEvent (%s), aborting...", err.Error())
		logger.Error(errMsg)
		return err
	}

	// Step 4 - prep-work
	// Get any additional input / configuration data
	// - Labels: get the incoming labels for potential config data and use it to pass more labels on result, e.g: links
	// - SLI.yaml: if your service uses SLI.yaml to store query definitions for SLIs get that file from Keptn
	labels := data.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	// Step 5 - get SLI Config File
	// Get SLI File from splunk subdirectory of the config repo - to add the file use:
	//   keptn add-resource --project=PROJECT --stage=STAGE --service=SERVICE --resource=my-sli-config.yaml  --resourceUri=splunk/sli.yaml
	sliConfig, err := ddKeptn.GetSLIConfiguration(data.Project, data.Stage, data.Service, SLI_FILE)

	// FYI you do not need to "fail" if sli.yaml is missing, you can also assume smart defaults like we do
	// in keptn-contrib/dynatrace-service and keptn-contrib/prometheus-service

	if err != nil {
		// failed to fetch sli config file
		errMsg := fmt.Sprintf("Failed to fetch SLI file %s from config repo: %s", SLI_FILE, err.Error())
		logger.Error(errMsg)
		// send a get-sli.finished event with status=error and result=failed back to Keptn

		_, _ = ddKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status: keptnv2.StatusErrored,
			Result: keptnv2.ResultFailed,
			Labels: labels,
		}, ServiceName)

		return err
	}
	logger.Infof("ResourceHandler : %v", ddKeptn.ResourceHandler)
	// Step 6 - do your work - iterate through the list of requested indicators and return their values
	// Indicators: this is the list of indicators as requested in the SLO.yaml
	// SLIResult: this is the array that will receive the results
	indicators := data.GetSLI.Indicators
	sliResults := []*keptnv2.SLIResult{}

	// get splunk API URL, PORT and TOKEN
	splunkCreds, err := getSplunkCredentials()
	if err != nil {
		logger.Errorf("failed to get Splunk Credentials: %v", err.Error())
		return err
	}

	logger.Info("indicators:", indicators)
	errored := false

	for _, indicatorName := range indicators {
		wg.Add(1)
		go handleSpecificSLI(indicatorName, splunkCreds, data, sliConfig, &sliResults, &errored)
		if errored {
			break
		}
	}

	wg.Wait()
	for _, sliResult := range sliResults {
		logger.Infof("SLI RESULTS for indicator %s : %v", sliResult.Metric, sliResult.Value)
	}

	// Step 7 - Build get-sli.finished event data
	getSliFinishedEventData := &keptnv2.GetSLIFinishedEventData{
		EventData: keptnv2.EventData{
			Status: keptnv2.StatusSucceeded,
			Result: keptnv2.ResultPass,
			Labels: labels,
		},
		GetSLI: keptnv2.GetSLIFinished{
			IndicatorValues: sliResults,
			Start:           data.GetSLI.Start,
			End:             data.GetSLI.End,
		},
	}

	if errored {
		getSliFinishedEventData.EventData.Status = keptnv2.StatusErrored
		getSliFinishedEventData.EventData.Result = keptnv2.ResultFailed
	}

	logger.Infof("SLI finished event: %v", *getSliFinishedEventData)

	_, err = ddKeptn.SendTaskFinishedEvent(getSliFinishedEventData, ServiceName)

	if err != nil {
		errMsg := fmt.Sprintf("Failed to send task finished CloudEvent (%s), aborting...", err.Error())
		logger.Error(errMsg)
		return err
	}

	return nil
}

func HandleConfigureMonitoringTriggeredEvent(ddKeptn *keptnv2.Keptn, incomingEvent cloudevents.Event, data *keptnv2.ConfigureMonitoringTriggeredEventData) error {
	var shkeptncontext string
	_ = incomingEvent.Context.ExtensionAs("shkeptncontext", &shkeptncontext)
	configureLogger(incomingEvent.Context.GetID(), shkeptncontext)

	logger.Infof("Handling configure-monitoring.triggered Event: %s", incomingEvent.Context.GetID())
	_, err := ddKeptn.SendTaskStartedEvent(data, ServiceName)
	if err != nil {
		logger.Errorf("err when sending task started the event: %v", err)
		return err
	}

	configureMonitoringFinishedEventData := &keptnv2.ConfigureMonitoringFinishedEventData{
		EventData: keptnv2.EventData{
			Status:  keptnv2.StatusSucceeded,
			Result:  keptnv2.ResultPass,
			Project: data.Project,
			Stage:   data.Service,
			Service: data.Service,
			Message: "Finished configuring monitoring",
		},
	}

	logger.Infof("Configure Monitoring finished event: %v", *configureMonitoringFinishedEventData)

	_, err = ddKeptn.SendTaskFinishedEvent(configureMonitoringFinishedEventData, ServiceName)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send task finished CloudEvent (%s), aborting...", err.Error())
		logger.Error(errMsg)
		return err
	}

	return nil
}

// getSplunkCredentials get the splunk host, port and api token from the environment variables set from secret
func getSplunkCredentials() (*splunkCredentials, error) {

	logger.Info("Trying to retrieve splunk credentials ...")

	splunkCreds := splunkCredentials{}

	if env.SplunkHost != "" && env.SplunkPort != "" && env.SplunkApiToken != "" {
		splunkCreds.Host = strings.Replace(env.SplunkHost, " ", "", -1)
		splunkCreds.Token = env.SplunkApiToken
		splunkCreds.Port = env.SplunkPort
		logger.Info("Successfully retrieved splunk credentials " + splunkCreds.Host + " and " + splunkCreds.Token + " and " + splunkCreds.Port)

	} else {
		logger.Info("SP_HOST, SP_PORT and/or SP_API_TOKEN have not correctly been set")
		return nil, errors.New("invalid credentials found in SP_HOST, SP_PORT and/or SP_API_TOKEN")
	}

	return &splunkCreds, nil
}

func handleSpecificSLI(indicatorName string, splunkCreds *splunkCredentials, data *keptnv2.GetSLITriggeredEventData, sliConfig map[string]string, sliResults *[]*keptnv2.SLIResult, errored *bool) {

	defer wg.Done()

	query := sliConfig[indicatorName]
	logger.Infof("actual query sent to splunk: %v, from: %v, to: %v", query, data.GetSLI.Start, data.GetSLI.End)

	if query == "" {
		*errored = true
		return
	}
	params := splunk.RequestParams{
		SearchQuery:  query,
		EarliestTime: data.GetSLI.Start,
		LatestTime:   data.GetSLI.End,
	}

	// no time range specified in the splunk search

	retrieveSearchTimeRange(&params)

	client := splunk.SplunkClient{
		Client: &http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		Host:  splunkCreds.Host,
		Port:  splunkCreds.Port,
		Token: splunkCreds.Token,
	}

	spReq := splunk.SplunkRequest{
		Params:  params,
		Headers: map[string]string{},
	}

	// get the metric we want
	sliValue, err := splunkjob.GetMetricFromNewJob(&client, &spReq)
	if err != nil {
		logger.Errorf("'%s': error getting value for the query: %v : %v\n", query, sliValue, err)
		*errored = true
		logger.WithFields(logger.Fields{"indicatorName": indicatorName}).Infof("got 0 in the SLI result (indicates empty response from the API)")

		return
	}

	logger.Infof("response from the metrics api: %v", sliValue)

	sliResult := &keptnv2.SLIResult{
		Metric:  indicatorName,
		Value:   sliValue,
		Success: true,
	}

	mutex.Lock()
	*sliResults = append(*sliResults, sliResult)
	mutex.Unlock()

	logger.WithFields(logger.Fields{"indicatorName": indicatorName}).Infof("SLI result from the metrics api: %v", sliResult)

}

// get the earliest and latest time from the splunk search
func retrieveSearchTimeRange(params *splunk.RequestParams) {

	// check if an earliest and/or latest time are set in the search
	search := params.SearchQuery
	queries := strings.Split(search, " ")
	start := params.EarliestTime
	end := params.LatestTime

	// TODO: don't go through all the query
	for _, q := range queries {

		if strings.HasPrefix(q, "earliest") && params.EarliestTime == start {
			params.EarliestTime = q[len("earliest")+1:]
		} else if strings.HasPrefix(q, "latest") && params.LatestTime == end {
			params.LatestTime = q[len("latest")+1:]
		}

		if params.EarliestTime != start && params.LatestTime != end {
			return
		}
	}
}
