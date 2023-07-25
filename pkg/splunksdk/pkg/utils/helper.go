package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

const JobsPathv2 = "services/search/v2/jobs/"
const SavedSearchesPath = "services/saved/searches/"
const GetAlertsNames = "getAlertsNames"
const GetTriggeredAlerts = "getTriggeredAlerts"
const CreateAlerts = "createAlerts"
const GetTriggeredInstances = "getTriggeredInstances"

// mock an http server
func MockRequest(response string, sslVerificationActivated bool) *httptest.Server {
	server := &httptest.Server{}
	if sslVerificationActivated {
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(response, &w, r)
		}))

	} else {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(response, &w, r)
		}))
	}
	return server
}

func MultitpleMockRequest(responses []map[string]interface{}, sslVerificationActivated bool) *httptest.Server {
	server := &httptest.Server{}
	if sslVerificationActivated {
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(responses, &w, r)
		}))
	} else {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(responses, &w, r)
		}))
	}
	return server
}

func writeResponses(responses interface{}, w *http.ResponseWriter, r *http.Request) {

	switch resps := responses.(type) {
	case []map[string]interface{}:
		for _, response := range resps {
			if r.Method == "GET" {
				if response[GetTriggeredAlerts] != nil && strings.HasSuffix(r.URL.Path, "services/alerts/fired_alerts/") {
					_, _ = (*w).Write([]byte(response[GetTriggeredAlerts].(string)))
				} else if response[GetTriggeredInstances] != nil && strings.Contains(r.URL.Path, "search/alerts/fired_alerts/") {
					_, _ = (*w).Write([]byte(response[GetTriggeredInstances].(string)))
				} else if response[GetAlertsNames] != nil && strings.Contains(r.URL.Path, "services/saved/searches/") {
					_, _ = (*w).Write([]byte(response[GetAlertsNames].(string)))
				} else if response["GET"] != nil && r.Method == "GET" {
					_, _ = (*w).Write([]byte(response["GET"].(string)))
				}
			}
			if r.Method == "POST" {
				if response[CreateAlerts] != nil && strings.Contains(r.URL.Path, "services/saved/searches/") {
					_, _ = (*w).Write([]byte(response[CreateAlerts].(string)))
				} else if response["POST"] != nil {
					_, _ = (*w).Write([]byte(response["POST"].(string)))
				}
			}
		}
	case string:
		_, _ = (*w).Write([]byte(resps))
	}
}

func GetTestHostname(server *httptest.Server) string {
	if os.Getenv("SPLUNK_HOST") != "" {
		return os.Getenv("SPLUNK_HOST")
	}
	return strings.Split(strings.Split(server.URL, ":")[1], "//")[1]

}

func GetTestPort(server *httptest.Server) string {
	if os.Getenv("SPLUNK_PORT") != "" {
		return os.Getenv("SPLUNK_PORT")
	}
	return strings.Split(server.URL, ":")[2]
}

func GetTestToken() string {
	if os.Getenv("SPLUNK_TOKEN") != "" {
		return os.Getenv("SPLUNK_TOKEN")
	}
	return "default"
}
