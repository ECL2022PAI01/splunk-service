package alerts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httputil"
	"strconv"
	"strings"

	utils "github.com/keptn-contrib/splunk-service/splunkSdkGo/pkg/utils"

	splunk "github.com/keptn-contrib/splunk-service/splunkSdkGo/src/client"
)

const savedSearchesPath = "services/saved/searches/"
const triggeredAlertsPath = "services/alerts/fired_alerts/"

type AlertRequest struct {
	Headers map[string]string
	Params  AlertParams
}

type AlertParams struct {
	Name         string
	Description  string `default:""`
	CronSchedule string
	// splunk search in spl syntax
	SearchQuery string
	OutputMode  string `default:"json"`
	// splunk returns a job SID only if the job is complete
	EarliestTime string
	// latest (exclusive) time bounds for the search
	LatestTime string
	//condition for triggering the alert
	AlertCondition      string
	AlertSuppress       string
	AlertSuppressPeriod string
	Actions             string
	WebhookUrl          string
}

type splunkAlertEntry struct {
	Name string `json:"name"`
}

type splunkAlertList struct {
	Item []splunkAlertEntry `json:"entry"`
}

type TriggeredAlerts struct {
	Origin  string      `json:"origin"`
	Updated string      `json:"updated"`
	Entry   []EntryItem `json:"entry"`
}

type TriggeredInstances struct {
	Origin  string      `json:"origin"`
	Updated string      `json:"updated"`
	Entry   []EntryItem `json:"entry"`
}

type EntryItem struct {
	Name    string  `json:"name"`
	Links   Links   `json:"links"`
	Content Content `content:"content"`
}

type Links struct {
	Alternate   string `json:"alternate"`
	List        string `json:"list"`
	Remove      string `json:"remove"`
	Job         string `json:"job"`
	SavedSearch string `json:"savedsearch"`
}

type Content struct {
	Sid                 string `json:"sid"`
	SavedSearchName     string `json:"savedsearch_name"`
	TriggerTime         int    `json:"trigger_time"`
	TriggeredAlertCount int    `json:"triggered_alert_count"`
}

// Creates a new alert from saved search
func CreateAlert(client *splunk.SplunkClient, spAlert *AlertRequest) error {

	// create the endpoint for the request
	utils.CreateEndpoint(client, savedSearchesPath)
	spAlert.Params.SearchQuery = utils.ValidateSearchQuery(spAlert.Params.SearchQuery)

	resp, err := PostAlert(client, spAlert)

	var respDump []byte
	var errDump error
	if resp != nil {
		respDump, errDump = httputil.DumpResponse(resp, true)
		if errDump != nil {
			fmt.Println(errDump)
		}
	}

	if err != nil {
		return fmt.Errorf("Alert creation : error while making the post request : %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	// handle error
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "2") {
		status, err := splunk.HandleHttpError(body)
		if err == nil {
			return fmt.Errorf("Alert creation : http error :  %s \nResponse : %v", status, string(respDump))
		} else {
			return fmt.Errorf("Alert creation : http error :  %s \nResponse : %v", resp.Status, string(respDump))
		}
	}

	if err != nil {
		return fmt.Errorf("Alert creation : error while getting the body of the post request : %s", err)
	}

	return nil
}

// Removes an existing saved search
func RemoveAlert(client *splunk.SplunkClient, alertName string) error {

	// create the endpoint for the request
	utils.CreateEndpoint(client, savedSearchesPath+alertName)

	splunkAlert := AlertRequest{}
	splunkAlert.Params.Name = alertName

	resp, err := DeleteAlert(client, &splunkAlert)

	var respDump []byte
	var errDump error
	if resp != nil {
		respDump, errDump = httputil.DumpResponse(resp, true)
		if errDump != nil {
			fmt.Println(errDump)
		}
	}

	if err != nil {
		return fmt.Errorf("Alert Removing : error while making the delete request : %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	// handle error
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "2") {
		status, err := splunk.HandleHttpError(body)
		if err == nil {
			return fmt.Errorf("Alert Removing : http error :  %s \nResponse : %v", status, string(respDump))
		} else {
			return fmt.Errorf("Alert Removing : http error :  %s \nResponse : %v", resp.Status, string(respDump))
		}
	}

	if err != nil {
		return fmt.Errorf("Alert Removing : error while getting the body of the delete request : %s", err)
	}

	return nil
}

// List saved searches
func ListAlertsNames(client *splunk.SplunkClient) (splunkAlertList, error) {

	var alertList splunkAlertList

	// create the endpoint for the request
	utils.CreateEndpoint(client, savedSearchesPath)

	resp, err := GetAlerts(client)

	var respDump []byte
	var errDump error
	if resp != nil {
		respDump, errDump = httputil.DumpResponse(resp, true)
		if errDump != nil {
			fmt.Println(errDump)
		}
	}

	if err != nil {
		return alertList, fmt.Errorf("Alerts' names listing : error while making the get request : %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	// handle error
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "2") {
		status, err := splunk.HandleHttpError(body)
		if err == nil {
			return alertList, fmt.Errorf("Alerts' names listing : http error :  %s \nResponse : %v", status, string(respDump))
		} else {
			return alertList, fmt.Errorf("Alerts' names listing : http error :  %s \nResponse : %v", resp.Status, string(respDump))
		}
	}

	if err != nil {
		return alertList, fmt.Errorf("Alerts' names listing : error while getting the body of the get request : %s", err)
	}

	err = json.Unmarshal(body, &alertList)
	if err != nil {
		return alertList, fmt.Errorf("Could not map list of alerts to datastructure: %s", err.Error())
	}

	return alertList, nil
}

func GetTriggeredAlerts(client *splunk.SplunkClient) (TriggeredAlerts, error) {

	var triggeredAlerts TriggeredAlerts

	// create the endpoint for the request
	utils.CreateEndpoint(client, triggeredAlertsPath)

	resp, err := GetAlerts(client)

	var respDump []byte
	var errDump error
	if resp != nil {
		respDump, errDump = httputil.DumpResponse(resp, true)
		if errDump != nil {
			fmt.Println(errDump)
		}
	}

	if err != nil {
		return triggeredAlerts, fmt.Errorf("Triggered alerts' names listing : error while making the get request : %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	// handle error
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "2") {
		status, err := splunk.HandleHttpError(body)
		if err == nil {
			return triggeredAlerts, fmt.Errorf("Triggered alerts' names listing : http error :  %s \nResponse : %v", status, string(respDump))
		} else {
			return triggeredAlerts, fmt.Errorf("Triggered alerts' names listing : http error :  %s \nResponse : %v", resp.Status, string(respDump))
		}
	}

	if err != nil {
		return triggeredAlerts, fmt.Errorf("Triggered alerts' names listing : error while getting the body of the get request : %s", err)
	}

	err = json.Unmarshal(body, &triggeredAlerts)
	if err != nil {
		return triggeredAlerts, fmt.Errorf("Could not map list of alerts to datastructure: %s", err.Error())
	}

	return triggeredAlerts, nil
}

func GetInstancesOfTriggeredAlert(client *splunk.SplunkClient, link string) (TriggeredInstances, error) {

	var triggeredInstances TriggeredInstances

	// create the endpoint for the request
	utils.CreateEndpoint(client, strings.TrimPrefix(link, "/"))

	resp, err := GetAlerts(client)

	var respDump []byte
	var errDump error
	if resp != nil {
		respDump, errDump = httputil.DumpResponse(resp, true)
		if errDump != nil {
			fmt.Println(errDump)
		}
	}

	if err != nil {
		return triggeredInstances, fmt.Errorf("Triggered instances' names listing : error while making the get request : %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	// handle error
	if !strings.HasPrefix(strconv.Itoa(resp.StatusCode), "2") {
		status, err := splunk.HandleHttpError(body)
		if err == nil {
			return triggeredInstances, fmt.Errorf("Triggered instances' names listing : http error :  %s \nResponse : %v, LINK : %v", status, string(respDump), client.Endpoint)
		} else {
			return triggeredInstances, fmt.Errorf("Triggered instances' names listing : http error :  %s \nResponse : %v, LINK : %v", status, string(respDump), client.Endpoint)
		}
	}

	if err != nil {
		return triggeredInstances, fmt.Errorf("Triggered instances' names listing : error while getting the body of the get request : %s", err)
	}

	err = json.Unmarshal(body, &triggeredInstances)
	if err != nil {
		return triggeredInstances, fmt.Errorf("Could not map list of alerts to datastructure: %s", err.Error())
	}

	return triggeredInstances, nil
}
