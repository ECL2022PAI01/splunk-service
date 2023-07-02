package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"gopkg.in/yaml.v2"

	"github.com/Mouhamadou305/splunk-service/pkg/utils"
	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	api "github.com/keptn/go-utils/pkg/api/utils"
	keptnevents "github.com/keptn/go-utils/pkg/lib"
	splunk "github.com/kuro-jojo/splunk-sdk-go/client"
	splunkjob "github.com/kuro-jojo/splunk-sdk-go/jobs"
	logger "github.com/sirupsen/logrus"
)

const sliFileUri = "splunk/sli.yaml"

type splunkCredentials struct {
	Host       string `json:"host" yaml:"spHost"`
	Port       string `json:"port" yaml:"spPort"`
	Username   string `json:"username" yaml:"spUsername"`
	Password   string `json:"password" yaml:"spPassword"`
	Token      string `json:"token" yaml:"spApiToken"`
	SessionKey string `json:"sessionKey" yaml:"spSessionKey"`
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
	sliConfig, err := ddKeptn.GetSLIConfiguration(data.Project, data.Stage, data.Service, sliFileUri)

	// FYI you do not need to "fail" if sli.yaml is missing, you can also assume smart defaults like we do
	// in keptn-contrib/dynatrace-service and keptn-contrib/splunk-service
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
	// Step 6 - do your work - iterate through the list of requested indicators and return their values
	// Indicators: this is the list of indicators as requested in the SLO.yaml
	// SLIResult: this is the array that will receive the results
	indicators := data.GetSLI.Indicators
	sliResults := []*keptnv2.SLIResult{}

	// get splunk API URL, PORT and TOKEN
	// TRY TO MAKE A FUNCTION
	splunkCreds, err := getSplunkCredentials()
	if err != nil {
		logger.Errorf("failed to get Splunk Credentials: %v", err.Error())
		return err
	}

	logger.Info("indicators:", indicators)
	var errSLI error
	var sliResult *keptnv2.SLIResult

	var client *splunk.SplunkClient
	if splunkCreds.Token != "" {
		client = splunk.NewClientAuthenticatedByToken(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.Token,
			true,
		)
	} else if splunkCreds.SessionKey != "" {
		client = splunk.NewClientAuthenticatedBySessionKey(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.SessionKey,
			true,
		)
	} else {
		client = splunk.NewBasicAuthenticatedClient(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.Username,
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

// Handles configure monitoring event
func HandleConfigureMonitoringTriggeredEvent(ddKeptn *keptnv2.Keptn, incomingEvent cloudevents.Event, data *keptnv2.ConfigureMonitoringTriggeredEventData) error {
	var shkeptncontext string
	logger.Info("We entered here")
	_ = incomingEvent.Context.ExtensionAs("shkeptncontext", &shkeptncontext)
	configureLogger(incomingEvent.Context.GetID(), shkeptncontext)

	logger.Infof("Handling configure-monitoring.triggered Event: %s", incomingEvent.Context.GetID())
	_, err := ddKeptn.SendTaskStartedEvent(data, ServiceName)
	if err != nil {
		logger.Errorf("err when sending task started the event: %v", err)
		return err
	}

	// --Start lets go

	// get splunk API URL, PORT and TOKEN
	splunkCreds, err := getSplunkCredentials()
	if err != nil {
		logger.Errorf("failed to get Splunk Credentials: %v", err.Error())
		return err
	}

	var client *splunk.SplunkClient
	if splunkCreds.Token != "" {
		client = splunk.NewClientAuthenticatedByToken(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.Token,
			true,
		)
	} else {
		client = splunk.NewBasicAuthenticatedClient(
			&http.Client{
				Timeout: time.Duration(60) * time.Second,
			},
			splunkCreds.Host,
			splunkCreds.Port,
			splunkCreds.Username,
			splunkCreds.Password,
			true,
		)
	}

	err = CreateSplunkAlertsForEachStage(client, ddKeptn, *data)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	// --End

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
	if env.SplunkHost != "" && env.SplunkPort != "" && (env.SplunkApiToken != "" || (env.SplunkUsername != "" && env.SplunkPassword != "") || env.SplunkSessionKey != "") {
		splunkCreds.Host = strings.ReplaceAll(env.SplunkHost, " ", "")
		splunkCreds.Token = env.SplunkApiToken
		splunkCreds.Port = env.SplunkPort
		splunkCreds.Username = env.SplunkUsername
		splunkCreds.Password = env.SplunkPassword
		splunkCreds.SessionKey = env.SplunkSessionKey

		logger.Info("Successfully retrieved splunk credentials")

	} else {
		logger.Info("SP_HOST, SP_PORT, SP_HOST, SP_API_TOKEN, SP_USERNAME, SP_PASSWORD and/or SP_SESSION_KEY have not correctly been set")
		return nil, errors.New("invalid credentials found in SP_HOST, SP_PORT, SP_HOST, SP_API_TOKEN, SP_USERNAME, SP_PASSWORD and/or SP_SESSION_KEY")
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

func CreateSplunkAlertsForEachStage(client *splunk.SplunkClient, k *keptnv2.Keptn, eventData keptnv2.ConfigureMonitoringTriggeredEventData) error {
	scope := api.NewResourceScope()
	scope.Project(eventData.Project)
	scope.Resource("shipyard.yaml")

	shipyard, err := k.GetShipyard()
	if err != nil {
		return err
	}

	for _, stage := range shipyard.Spec.Stages {
		err = CreateSplunkAlertsIfSLOsAndRemediationDefined(client, k, eventData, stage)

		if err != nil {
			return fmt.Errorf("error configuring splunk alerts: %w", err)
		}
	}

	return nil

}

func CreateSplunkAlertsIfSLOsAndRemediationDefined(client *splunk.SplunkClient, k *keptnv2.Keptn, eventData keptnv2.ConfigureMonitoringTriggeredEventData, stage keptnv2.Stage) error {
	logger.Info("We entered here")
	slos, err := retrieveSLOs(k.ResourceHandler, eventData, stage.Name)
	if err != nil || slos == nil {
		logger.Info("No SLO file found for stage " + stage.Name + ". No alerting rules created for this stage")
		return nil //SHOULD BE NIL
	}

	const remediationFileDefaultName = "remediation.yaml"

	resourceScope := api.NewResourceScope()
	resourceScope.Project(eventData.Project)
	resourceScope.Service(eventData.Service)
	resourceScope.Stage(stage.Name)
	resourceScope.Resource(remediationFileDefaultName)

	_, err = k.ResourceHandler.GetResource(*resourceScope)

	if errors.Is(err, api.ResourceNotFoundError) {
		logger.Infof("No remediation defined for project %s stage %s, skipping setup of prometheus alerts",
			eventData.Project, stage.Name)
		return nil //SHOULD BE NIL
	}

	if err != nil {
		return fmt.Errorf("error retrieving remediation definition %s for project %s and stage %s: %w",
			remediationFileDefaultName, eventData.Project, stage.Name, err)
	}

	// get SLI queries
	projectCustomQueries, err := getCustomQueries(k, eventData.Project, stage.Name, eventData.Service)
	if err != nil {
		log.Println("Failed to get custom queries for project " + eventData.Project)
		log.Println(err.Error())
		return err
	}

	logger.Info("Going over SLO.objectives")

	for _, objective := range slos.Objectives {
		logger.Info("SLO:" + objective.DisplayName + ", " + objective.SLI)

		end := "now"
		start := "-3m"

		query := projectCustomQueries[objective.SLI]

		if err != nil || query == "" {
			logger.Error("No query defined for SLI " + objective.SLI + " in project " + eventData.Project)
			continue
		}
		logger.Info("query= " + query)

		resultField, err := getResultFieldName(query)
		if err != nil {
			log.Println("Failed to get the result field name in order to create the alert condition for " + eventData.Project)
			log.Println(err.Error())
			return err
		}

		if objective.Pass != nil {
			for _, criteriaGroup := range objective.Pass {
				for _, criteria := range criteriaGroup.Criteria {

					//TO SUPPORT RELATIVE CRITERIAS I'LL HAVE TO MODIFY THAT PART
					if strings.Contains(criteria, "+") || strings.Contains(criteria, "-") || strings.Contains(
						criteria, "%",
					) || (!strings.Contains(criteria, "<") && !strings.Contains(criteria, ">")) {
						continue
					}
					//

					if strings.Contains(criteria, "<=") {
						criteria = strings.Replace(criteria, "<=", ">", -1)
					} else if strings.Contains(criteria, "<") {
						criteria = strings.Replace(criteria, "<", ">=", -1)
					} else if strings.Contains(criteria, ">=") {
						criteria = strings.Replace(criteria, ">=", "<", -1)
					} else {
						criteria = strings.Replace(criteria, ">", "<=", -1)
					}

					// sanitize criteria : remove whitespaces
					criteria = strings.Replace(criteria, " ", "", -1)

					alertCondition := buildAlertCondition(resultField, criteria)
					alertName := buildAlertName(eventData, stage.Name, objective.SLI, criteria)
					cronSchedule := "*/1 * * * *"
					actions := "logevent,webhook"
					webhookUrl := "localhost:8080" //WARNING CHANGE THIS

					params := splunk.AlertParams{
						Name:           alertName,
						CronSchedule:   cronSchedule,
						SearchQuery:    query,
						EarliestTime:   start,
						LatestTime:     end,
						AlertCondition: alertCondition,
						Actions:        actions,
						WebhookUrl:     webhookUrl,
					}
					utils.RetrieveAlertTimeRange(&params)

					spAlert := splunk.SplunkAlert{
						Params:  params,
						Headers: map[string]string{},
					}

					// get the metric we want
					err := splunkjob.CreateAlert(client, &spAlert)
					if err != nil {
						//DONT FORGET TO TAKE INTO ACCOUNT THE FACT THAT THE ERROR COULD BE CAUSED BY THE FACT
						//THAT TH ALERT ALREADY EXIST. IN THAT CASE WE SHOULD UPDATE THE ALERT
						logger.Errorf("Error calling CreateAlert(): %v : %v :\n %v", spAlert.Params.SearchQuery, err, client)
					}

				}
			}
		}
	}
	return nil
}

func retrieveSLOs(resourceHandler *api.ResourceHandler, eventData keptnv2.ConfigureMonitoringTriggeredEventData, stage string) (*keptnevents.ServiceLevelObjectives, error) {
	resourceScope := api.NewResourceScope()
	resourceScope.Project(eventData.Project)
	resourceScope.Service(eventData.Service)
	resourceScope.Stage(stage)
	resourceScope.Resource("slo.yaml")

	resource, err := resourceHandler.GetResource(*resourceScope)
	if err != nil || resource.ResourceContent == "" {
		return nil, errors.New("No SLO file available for service " + eventData.Service + " in stage " + stage)
	}
	var slos keptnevents.ServiceLevelObjectives

	err = yaml.Unmarshal([]byte(resource.ResourceContent), &slos)

	if err != nil {
		return nil, errors.New("Invalid SLO file format")
	}

	return &slos, nil
}

func getCustomQueries(k *keptnv2.Keptn, project string, stage string, service string) (map[string]string, error) {
	log.Println("Checking for custom SLI queries")

	customQueries, err := k.GetSLIConfiguration(project, stage, service, sliFileUri)
	if err != nil {
		return nil, err
	}

	return customQueries, nil
}

func getResultFieldName(searchQuery string) (string, error) {
	if strings.Contains(searchQuery, "stats") {
		startIndex := strings.Index(searchQuery, "stats")+4
		i := 1
		for {
			if searchQuery[startIndex+i] == " "[0] && searchQuery[startIndex+i-1] != "s"[0] {
				return searchQuery[startIndex+2 : startIndex+i], nil
			}
			i = i + 1
			if i == len(searchQuery) {
				break
			}
		}
	}
	return "", errors.New("No aggragation function found in the search query.")
}

func buildAlertCondition(resultField string, criteria string) string {
	return "search " + resultField + " " + criteria
}

func buildAlertName(eventData keptnv2.ConfigureMonitoringTriggeredEventData, stage string, sli string, criteria string) string {
	return eventData.Project + "_" + stage + "_" + eventData.Service + "_" + sli + "_" + criteria
}
