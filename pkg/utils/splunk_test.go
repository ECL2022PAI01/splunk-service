package utils

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// Tests the getSplunkCredentials function
func TestGetSplunkCredentials(t *testing.T) {
	godotenv.Load(".env.local")

	env := EnvConfig{}
	env.SplunkApiToken = os.Getenv("SPLUNK_API_TOKEN")
	env.SplunkHost = os.Getenv("SPLUNK_HOST")
	env.SplunkPort = os.Getenv("SPLUNK_PORT")

	sp, err := GetSplunkCredentials(env)

	if err == nil {
		t.Logf("Splunk credentials : %v", sp)
		if sp.Host == "" || sp.Port == "" || sp.Token == "" {
			t.Fatal("If Host, Port or token are empty. An error should be returned")
		}
	} else {
		t.Logf("Received expected error : %s", err.Error())
	}
}
