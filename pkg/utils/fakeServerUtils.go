package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

func MultitpleMockRequest(getResponses []string, postResponses []string, paths []string, sslVerificationActivated bool) *httptest.Server {
	server := &httptest.Server{}
	if sslVerificationActivated {
		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(getResponses, postResponses, &w, r, paths)
		}))
	} else {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			writeResponses(getResponses, postResponses, &w, r, paths)
		}))
	}
	return server
}

func writeResponses(getResponses []string, postResponses []string, w *http.ResponseWriter, r *http.Request, paths []string) {

	if(r.Method=="GET"){
		for i, response := range getResponses {
				if response != "" && strings.HasSuffix(r.URL.Path, paths[i]) {
					_, _ = (*w).Write([]byte(response))
			}
		}
	} else if(r.Method=="POST"){
		for i, response := range postResponses {
				if response != "" && strings.HasSuffix(r.URL.Path, paths[i]) {
					_, _ = (*w).Write([]byte(response))
			}
		}
	}

}