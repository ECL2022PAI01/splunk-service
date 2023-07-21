package jobs

import (
	"net/http"
	"net/url"

	utils "github.com/ECL2022PAI01/splunk-service/splunkSdkGo/pkg/utils"

	splunk "github.com/ECL2022PAI01/splunk-service/splunkSdkGo/src/client"
)

func PostJob(client *splunk.SplunkClient, spRequest *SearchRequest) (*http.Response, error) {

	return HttpJobRequest(client, "POST", spRequest)
}

func GetJob(client *splunk.SplunkClient) (*http.Response, error) {

	return HttpJobRequest(client, "GET", nil)
}

func HttpJobRequest(client *splunk.SplunkClient, method string, spRequest *SearchRequest) (*http.Response, error) {

	if spRequest == nil {
		spRequest = &SearchRequest{}
	}

	spRequest.Params.OutputMode = "json"
	spRequest.Params.ExecMode = "blocking"

	// parameters of the request
	params := url.Values{}
	params.Add("output_mode", spRequest.Params.OutputMode)
	params.Add("exec_mode", spRequest.Params.ExecMode)

	if method == "POST" {
		params.Add("search", utils.ValidateSearchQuery(spRequest.Params.SearchQuery))
		if spRequest.Params.EarliestTime != "" {
			params.Add("earliest_time", spRequest.Params.EarliestTime)
		}
		if spRequest.Params.LatestTime != "" {
			params.Add("latest_time", spRequest.Params.LatestTime)
		}
	}

	return splunk.MakeHttpRequest(client, method, spRequest.Headers, params)
}
