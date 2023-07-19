package alerts

import (
	"strings"
	"testing"

	"github.com/ECL2022PAI01/splunk-service/pkg/utils"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"
)

func TestFiringAlertsPoll(t *testing.T) {
	//Building a mock splunk server
	splunkServer := utils.BuildMockSplunkServer(0)
	defer splunkServer.Close()

	//setting splunk credentials
	env := utils.EnvConfig{}
	env.SplunkPort = strings.Split(splunkServer.URL, ":")[2]
	env.SplunkHost = strings.Split(strings.Split(splunkServer.URL, ":")[1], "//")[1]
	env.SplunkApiToken = "apiToken"
	keptnOptions := keptn.KeptnOpts{
		EventSender:             &fake.EventSender{},
		ConfigurationServiceURL: splunkServer.URL + "/api/resource-service",
		UseLocalFileSystem:      true,
	}
	go FiringAlertsPoll(keptnOptions, env)

}
