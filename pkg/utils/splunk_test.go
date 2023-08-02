package utils

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// Tests the getSplunkCredentials function
func TestGetSplunkCredentials(t *testing.T) {
	err := godotenv.Load(".env.local")
	if err != nil {
		t.Fatalf("Got an error loading .env.local file: %s", err)
	}

	env := EnvConfig{}
	env.SplunkApiToken = os.Getenv("SPLUNK_API_TOKEN")
	env.SplunkHost = os.Getenv("SPLUNK_HOST")
	env.SplunkPort = os.Getenv("SPLUNK_PORT")

	sp, err := GetSplunkCredentials(env)

	switch err {
	case nil:
		t.Logf("Splunk credentials : %v", sp)
		if sp.Host == "" || sp.Port == "" || sp.Token == "" {
			t.Fatal("If Host, Port or token are empty. An error should be returned")
		}
	default:
		t.Logf("Received expected error : %v", err)
	}
}
