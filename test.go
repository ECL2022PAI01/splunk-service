package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("python", "-c", "import splunk; print(splunk.SplunkProvider(project='test-splunk',stage='qa',service='helloservice', labels={}, customQueries={\"test_query\" : \"search |inputcsv test.csv | stats count\"}, host='localhost', token='eyJraWQiOiJzcGx1bmsuc2VjcmV0IiwiYWxnIjoiSFM1MTIiLCJ2ZXIiOiJ2MiIsInR0eXAiOiJzdGF0aWMifQ.eyJpc3MiOiJhZG1pbiBmcm9tIGJhMzljNjk3ZTA5ZCIsInN1YiI6ImFkbWluIiwiYXVkIjoidGVzdCIsImlkcCI6IlNwbHVuayIsImp0aSI6IjU4MTRjNjBmNDNlNzk5ZDI1YzEzZDMyOWE4NTY2ZGM0ZmM5Mjg4MjQyMTg0NTAwMDY1NTdhYTYyYTI0YzYyNjQiLCJpYXQiOjE2Nzk0MTQxMjIsImV4cCI6MTY4MDQ1MDkyMiwibmJyIjoxNjc5NDE0MTIyfQ.gK2mdx7X8L6sdi50E0RvEI7wAvjEdq1P489pQ1isIRF5TzbL_RIXoB0Ku-zeRmo_Wc8hAcSNDPftu8QUuBCRkA', port=8000).get_sli('test_query', '2023-03-21T22:00:43.940Z','2023-03-21T22:02:43.940Z'))")
	fmt.Println(cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(out))
}
