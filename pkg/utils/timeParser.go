package utils

import (
	"strings"

	splunkalert "github.com/kuro-jojo/splunk-sdk-go/src/alerts"
	splunkjob "github.com/kuro-jojo/splunk-sdk-go/src/jobs"
)

// check if the search string contains the earliest or latest time and return the value
func getSearchTime(kind string, search string, params *splunkjob.SearchParams, defaultTime string) string {
	if strings.Contains(search, kind) {
		startIndex := strings.Index(search, kind)
		q1 := strings.Fields(search[startIndex:])

		val := ""
		if !strings.HasPrefix(q1[0][len(kind)+1:], "\"") {
			val = q1[0][len(kind)+1:]
			params.SearchQuery = strings.ReplaceAll(params.SearchQuery, q1[0], "")
		} else {
			for i, v := range q1 {
				if i == 0 {
					val += v[len(kind)+2:]
				} else {
					val += " " + v
				}
				if strings.HasSuffix(v, "\"") {
					break
				}
			}
			params.SearchQuery = strings.ReplaceAll(params.SearchQuery, val, "")
			params.SearchQuery = strings.ReplaceAll(params.SearchQuery, "earliest=\"", "")
		}
		return strings.TrimSuffix(val, "\"")
	}

	return defaultTime
}

// check if the search string contains the earliest or latest time and return the value
func getAlertTime(kind string, search string, params *splunkalert.AlertParams, defaultTime string) string {
	if strings.Contains(search, kind) {
		startIndex := strings.Index(search, kind)
		q1 := strings.Fields(search[startIndex:])

		val := ""
		if !strings.HasPrefix(q1[0][len(kind)+1:], "\"") {
			val = q1[0][len(kind)+1:]
			params.SearchQuery = strings.ReplaceAll(params.SearchQuery, q1[0], "")
		} else {
			for i, v := range q1 {
				if i == 0 {
					val += v[len(kind)+2:]
				} else {
					val += " " + v
				}
				if strings.HasSuffix(v, "\"") {
					break
				}
			}
			params.SearchQuery = strings.ReplaceAll(params.SearchQuery, val, "")
			params.SearchQuery = strings.ReplaceAll(params.SearchQuery, "earliest=\"", "")
		}
		return strings.TrimSuffix(val, "\"")
	}

	return defaultTime
}

// get the earliest and latest time from the splunk search is set
func RetrieveSearchTimeRange(params *splunkjob.SearchParams) {
	search := params.SearchQuery

	params.EarliestTime = getSearchTime("earliest", search, params, params.EarliestTime)
	params.LatestTime = getSearchTime("latest", search, params, params.LatestTime)
}

func RetrieveAlertTimeRange(params *splunkalert.AlertParams) {
	search := params.SearchQuery

	params.EarliestTime = getAlertTime("earliest", search, params, params.EarliestTime)
	params.LatestTime = getAlertTime("latest", search, params, params.LatestTime)
}
