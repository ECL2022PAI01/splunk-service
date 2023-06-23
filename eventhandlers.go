package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	splunk "github.com/kuro-jojo/splunk-sdk-go/client"
	splunkjob "github.com/kuro-jojo/splunk-sdk-go/jobs"
	"github.com/kuro-jojo/splunk-service/pkg/utils"
	logger "github.com/sirupsen/logrus"
)



type splunkCredentials struct {
	Host  		string `json:"host" yaml:"spHost"`
	Token 		string `json:"token" yaml:"spToken"`
	Port  		string `json:"port" yaml:"spPort"`
	User  		string `json:"user" yaml:"spUser"`
	Password  	string `json:"password" yaml:"spPassword"`
}

// HandleGetSliTriggeredEvent handles get-sli.triggered events if SLIProvider == splunk
func HandleGetSliTriggeredEvent(ddKeptn *keptnv2.Keptn, incomingEvent cloudevents.Event, data *keptnv2.GetSLITriggeredEventData) error {
	const sliFileUri = "splunk/sli.yaml"
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
	sliConfig, err := ddKeptn.GetSLIConfiguration(data.Project, data.Stage, data.Service, sliFileUri)

	// FYI you do not need to "fail" if sli.yaml is missing, you can also assume smart defaults like we do
	// in keptn-contrib/dynatrace-service and keptn-contrib/prometheus-service
	logger.Infof("SLI Config: %s", sliConfig)
	if err != nil {
		// failed to fetch sli config file
		errMsg := fmt.Sprintf("Failed to fetch SLI file %s from config repo: %s", sliFileUri, err.Error())
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
	var errSLI error
	var sliResult *keptnv2.SLIResult

	var client *splunk.SplunkClient
	if(splunkCreds.Token!=""){
		client = splunk.NewClientAuthenticatedByToken(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.Token,
			true,
		)
	}else{
		client = splunk.NewBasicAuthenticatedClient(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.User,
			splunkCreds.Password,
			true,
		)
	}

	for _, indicatorName := range indicators {
		sliResult, errSLI = handleSpecificSLI(client, indicatorName, data, sliConfig)
		if errSLI != nil {
			break
		}

		sliResults = append(sliResults, sliResult)
	}

	logger.Infof("SLI Results: %v", sliResults)
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

	if errSLI != nil {
		getSliFinishedEventData.EventData.Status = keptnv2.StatusErrored
		getSliFinishedEventData.EventData.Result = keptnv2.ResultFailed
		getSliFinishedEventData.EventData.Message = fmt.Sprintf("error from the %s while getting slis : %s", ServiceName, errSLI.Error())
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

//Handles configure monitoring event
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

	if env.SplunkHost != "" && env.SplunkPort != "" && ( env.SplunkApiToken != "" || (env.SplunkUser!="" && env.SplunkPassword!="") ) {
		splunkCreds.Host = strings.Replace(env.SplunkHost, " ", "", -1)
		splunkCreds.Token = env.SplunkApiToken
		splunkCreds.Port = env.SplunkPort
		splunkCreds.User = env.SplunkUser
		splunkCreds.Password = env.SplunkPassword
		logger.Info("Successfully retrieved splunk credentials")

	} else {
		logger.Info("SP_HOST, SP_PORT, SP_API_TOKEN, SP_HOST and/or SP_PASSWORD have not correctly been set")
		return nil, errors.New("Invalid credentials found in SP_HOST, SP_PORT, SP_API_TOKEN, SP_HOST and/or SP_PASSWORD")
	}

	return &splunkCreds, nil
}

func handleSpecificSLI(client *splunk.SplunkClient, indicatorName string, data *keptnv2.GetSLITriggeredEventData, sliConfig map[string]string) (*keptnv2.SLIResult, error) {

	query := sliConfig[indicatorName]
	params := splunk.RequestParams{
		SearchQuery:  query,
		EarliestTime: data.GetSLI.Start,
		LatestTime:   data.GetSLI.End,
	}

	// take the time range from the sli file if it is set
	utils.RetrieveSearchTimeRange(&params)
	logger.Infof("actual query sent to splunk: %v, from: %v, to: %v", params.SearchQuery, params.EarliestTime, params.LatestTime)

	if query == "" {
		return nil, fmt.Errorf("no query found for indicator %s", indicatorName)
	}

	spReq := splunk.SplunkRequest{
		Params:  params,
		Headers: map[string]string{},
	}

	// get the metric we want
	sliValue, err := splunkjob.GetMetricFromNewJob(client, &spReq)
	if err != nil {
		return nil, fmt.Errorf("error getting value for the query: %v : %v", spReq.Params.SearchQuery, err)
	}

	logger.Infof("response from the metrics api: %v", sliValue)

	sliResult := &keptnv2.SLIResult{
		Metric:  indicatorName,
		Value:   sliValue,
		Success: true,
	}
	logger.WithFields(logger.Fields{"indicatorName": indicatorName}).Infof("SLI result from the metrics api: %v", sliResult)
	
	return sliResult, nil
}
