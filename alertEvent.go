package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	api "github.com/keptn/go-utils/pkg/api/utils"
	keptncommons "github.com/keptn/go-utils/pkg/lib"

	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

const remediationTaskName = "remediation"

type splunkAlertEvent struct {
	Sid string  `json:"sid"`
	SearchName   string  `json:"search_name"`
	App string `json:"app"`
	Owner string `json:"owner"`
	ResultsLink string `json:"results_link"`
	Result   alertResult `json:"result"`
}

// alert coming from splunk
type alertResult struct {
	Avg     			string      `json:"avg"`
	Count       		string      `json:"count"`
	DistinctCount       string      `json:"distinct_count"`
	Estdc 				string      `json:"estdc"`
	EstdcError 			string      `json:"estdc_error"`
	Exactperc 			string      `json:"exactperc"`
	Max 				string      `json:"max"`
	Mean 				string      `json:"mean"`
	Median 				string      `json:"median"`
	Min 				string      `json:"min"`
	Mode 				string      `json:"mode"`
	Perc 				string      `json:"perc"`
	Range 				string      `json:"range"`
	Stdev 				string      `json:"stdev"`
	Stdevp 				string      `json:"stdevp"`
	Sum 				string      `json:"sum"`
	Sumsq 				string      `json:"sumsq"`
	Upperperc 			string      `json:"upperperc"`
	Var 				string      `json:"var"`
	Varp 				string      `json:"varp"`
}

type labels struct {
	AlertName  string `json:"alertname,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	PodName    string `json:"pod_name,omitempty"`
	Severity   string `json:"severity,omitempty"`
	Service    string `json:"service,omitempty" yaml:"service"`
	Stage      string `json:"stage,omitempty" yaml:"stage"`
	Project    string `json:"project,omitempty" yaml:"project"`
	Deployment string `json:"deployment,omitempty" yaml:"deployment"`
}


type remediationTriggeredEventData struct {
	keptnv2.EventData

	// Problem contains details about the problem
	Problem keptncommons.ProblemEventData `json:"problem"`
	// Deployment contains the current deployment, that is inferred from the alert event

	Deployment keptnv2.DeploymentFinishedData `json:"deployment"`
}

// ProcessAndForwardAlertEvent reads the payload from the request and sends a valid Cloud event to the keptn event broker
func ProcessAndForwardAlertEvent(rw http.ResponseWriter, requestBody []byte, logger *keptn.Logger, shkeptncontext string) {
	var event splunkAlertEvent

	logger.Info("Received alert from Splunk Alerting system : " + string(requestBody))
	err := json.Unmarshal(requestBody, &event)
	if err != nil {
		logger.Error("Could not map received event to datastructure: " + err.Error())
		return
	}

	const deploymentType = "primary"
	alertDetails := strings.Split(event.SearchName, ",")

	problemData := keptncommons.ProblemEventData{
		State:          "OPEN",
		ProblemID:      "",
		ProblemTitle:   alertDetails[3],    //name of sli
		ProblemDetails: json.RawMessage(`{}`),
		ProblemURL:     event.ResultsLink,
		ImpactedEntity: fmt.Sprintf("%s-%s", alertDetails[2], deploymentType),
		Project:        alertDetails[0],
		Stage:          alertDetails[1],
		Service:        alertDetails[2],
		Labels: map[string]string{
			"deployment": deploymentType,
		},
	}

	newEventData := remediationTriggeredEventData{
		EventData: keptnv2.EventData{
			Project:        alertDetails[0],
			Stage:          alertDetails[1],
			Service:        alertDetails[2],
			Labels: map[string]string{
				"Problem URL": event.ResultsLink,
			},
		},
		Problem: problemData,
		Deployment: keptnv2.DeploymentFinishedData{
			DeploymentNames: []string{
				deploymentType,
			},
		},
	}

	if event.Sid != "" {
		shkeptncontext = createOrApplyKeptnContext(event.Sid + time.Now().Format(time.UnixDate))
		logger.Debug("shkeptncontext=" + shkeptncontext)
	} else {
		logger.Debug("NO SHKEPTNCONTEXT SET")
	}

	logger.Debug("Sending event to eventbroker")
	err = createAndSendCE(newEventData, shkeptncontext)
	if err != nil {
		logger.Error("Could not send cloud event: " + err.Error())
		rw.WriteHeader(500)
	} else {
		logger.Debug("event successfully dispatched to eventbroker")
		rw.WriteHeader(201)
	}
}

// createAndSendCE create a new problem.triggered event and send it to Keptn
func createAndSendCE(problemData remediationTriggeredEventData, shkeptncontext string) error {
	source, _ := url.Parse("splunk")

	eventType := keptnv2.GetTriggeredEventType(problemData.Stage + "." + remediationTaskName)

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetTime(time.Now())
	event.SetType(eventType)
	event.SetSource(source.String())
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetExtension("shkeptncontext", shkeptncontext)
	err := event.SetData(cloudevents.ApplicationJSON, problemData)
	if err != nil {
		return fmt.Errorf("unable to set cloud event data: %w", err)
	}

	ddKeptn, err := keptnv2.NewKeptn(&event, keptnOptions)

	//Setting authentication header when accessing to keptn locally in order to be able to access to the resource-service
	if env.Env == "local" {
		authToken := os.Getenv("KEPTN_API_TOKEN")
		authHeader := "x-token"
		ddKeptn.ResourceHandler = api.NewAuthenticatedResourceHandler(ddKeptn.ResourceHandler.BaseURL, authToken, authHeader, ddKeptn.ResourceHandler.HTTPClient, ddKeptn.ResourceHandler.Scheme)
	}

	if err != nil {
		return fmt.Errorf("Could not create Keptn Handler: " + err.Error())
	}

	err = ddKeptn.SendCloudEvent(event)
	if err != nil {
		return err
	}

	return nil
}

// createOrApplyKeptnContext re-uses the existing Keptn Context or creates a new one based on splunk fingerprint
func createOrApplyKeptnContext(contextID string) string {
	uuid.SetRand(nil)
	keptnContext := uuid.New().String()
	if contextID != "" {
		_, err := uuid.Parse(contextID)
		if err != nil {
			if len(contextID) < 16 {
				// use provided contxtId as a seed
				paddedContext := fmt.Sprintf("%-16v", contextID)
				uuid.SetRand(strings.NewReader(paddedContext))
			} else {
				// convert hash of contextID
				h := sha256.New()
				h.Write([]byte(contextID))
				bs := h.Sum(nil)

				uuid.SetRand(strings.NewReader(string(bs)))
			}

			keptnContext = uuid.New().String()
			uuid.SetRand(nil)
		} else {
			keptnContext = contextID
		}
	}
	return keptnContext
}