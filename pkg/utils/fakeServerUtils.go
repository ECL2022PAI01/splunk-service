package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

func MultitpleMockRequest(getResponses []string, postResponses []string, paths []string, sslVerificationActivated bool) *httptest.Server {
	var server *httptest.Server
	switch sslVerificationActivated {
	case true:
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(getResponses, postResponses, w, r, paths)
		}))
	default:
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(getResponses, postResponses, w, r, paths)
		}))
	}
	return server
}

func writeResponses(getResponses []string, postResponses []string, w http.ResponseWriter, r *http.Request, paths []string) {

	switch method := r.Method; method {
	case "GET":
		for i, response := range getResponses {
			if response != "" && strings.HasSuffix(r.URL.Path, paths[i]) {
				_, _ = (w).Write([]byte(response))
			}
		}
	case "POST":
		for i, response := range postResponses {
			if response != "" && strings.HasSuffix(r.URL.Path, paths[i]) {
				_, _ = (w).Write([]byte(response))
			}
		}
	}

}
