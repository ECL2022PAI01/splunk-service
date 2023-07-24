package tests

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/keptn-contrib/splunk-service/splunkSdkGo/pkg/utils"

	splunk "github.com/keptn-contrib/splunk-service/splunkSdkGo/src/client"

	"github.com/keptn-contrib/splunk-service/splunkSdkGo/src/jobs"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	initialize()
	code := m.Run()

	os.Exit(code)
}

func initialize() {
	godotenv.Load(".env")
	if !RunTestsWithEnvVars() {
		return
	}
}

func RunTestsWithEnvVars() bool {
	return os.Getenv("SPLUNK_ENV") == "LOCAL"
}

func TestGetMetric(t *testing.T) {
	jsonResponsePOST := `{
		"sid": "1689673231.191"
	}`

	jsonResponseGET := `{
		"results":[{"count":"2566"}]
	}`

	responses := make([]map[string]interface{}, 2)
	responses[0] = map[string]interface{}{
		"POST": jsonResponsePOST,
	}
	responses[1] = map[string]interface{}{
		"GET": jsonResponseGET,
	}

	server := MultitpleMockRequest(responses, true)

	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		GetTestHostname(server),
		GetTestPort(server),
		GetTestToken(),
		true,
	)

	defer server.Close()

	spReq := jobs.SearchRequest{
		Params: jobs.SearchParams{
			SearchQuery: "source=\"http:podtato-error\" (index=\"keptn-splunk-dev\") \"[error]\" | stats count",
		},
	}

	metric, err := jobs.GetMetricFromNewJob(client, &spReq)

	if err != nil {
		t.Fatalf("Got an error : %s", err)
	}

	expectedMetric := 2566
	if metric != float64(expectedMetric) {
		t.Fatalf("Expected %v but got %v.", expectedMetric, metric)
	}
}

func TestCreateJob(t *testing.T) {

	jsonResponsePOST := `{
		"sid": "1689673231.191"
	}`
	server := MockRequest(jsonResponsePOST, true)
	defer server.Close()

	spReq := jobs.SearchRequest{
		Params: jobs.SearchParams{
			SearchQuery: "source=\"http:podtato-error\" (index=\"keptn-splunk-dev\") \"[error]\" | stats count",
		},
	}
	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		GetTestHostname(server),
		GetTestPort(server),
		GetTestToken(),
		true,
	)

	utils.CreateEndpoint(client, jobsPathv2)

	sid, err := jobs.CreateJob(client, &spReq, jobsPathv2)

	if err != nil {
		t.Fatalf("Got an error : %s", err)
	}

	expectedSID := "1689673231.191"
	if sid != expectedSID {
		t.Fatalf("Expected %v but got %v.", expectedSID, sid)
	}
}

func TestRetrieveJobResult(t *testing.T) {

	jsonResponseGET := `{
		"results":[{"count":"2566"}]
	}`
	server := MockRequest(jsonResponseGET, true)
	defer server.Close()

	client := splunk.NewClientAuthenticatedByToken(
		&http.Client{
			Timeout: time.Duration(60) * time.Second,
		},
		GetTestHostname(server),
		GetTestPort(server),
		GetTestToken(),
		true,
	)
	utils.CreateEndpoint(client, jobsPathv2)
	results, err := jobs.RetrieveJobResult(client, "1689673231.191")

	if err != nil {
		t.Fatalf("Got an error : %s", err)
	}

	expectedRes := make([]map[string]string, 1)
	expectedRes[0] = map[string]string{
		"count": "2566",
	}

	if results[0]["count"] != expectedRes[0]["count"] {
		t.Fatalf("Expected %v but got %v.", expectedRes, results)
	}
}
