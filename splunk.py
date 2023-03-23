from logging import error
from typing import Dict, Tuple
import math
from datetime import datetime
import splunklib.client as client
import splunklib.results as results

class SplunkProvider:
    def __init__(
        self,
        project,
        stage,
        service,
        labels: Dict[str, str] = None,
        customQueries: Dict[str, str] = None,
        host: str="",
        username: str="",
        password: str="",
        token: str="",
        autologin= True,
        port: str="8089"
    ) -> None:
        if(username!="" and password!=""):
            #connecting using username and password
            self.splunkService = client.connect(host=host, port=int(port), username=username, password=password, autologin=autologin)
        elif(token!="" and host!=""):
            #connecting using bearer token
            self.splunkService = client.connect(host=host, port=int(port), splunkToken=token, autologin=autologin)
        else:
            error("Connection credentials are invalid")

        self.project = project
        self.stage = stage
        self.service= service
        self.labels = labels
        self.custom_queries = customQueries

    def get_sli(
        self, metric: str, start_time: str, end_time: str
    ) -> Tuple[float, None]:
        start_unix, end_unix = self._parse_unix_timestamps(start_time, end_time)

        kwargs_oneshot = {"earliest_time": start_time,
                  "latest_time": end_time,
                  "output_mode": 'json'}
        searchquery_oneshot = self._get_metric_query(metric, start_unix, end_unix)

        oneshotsearch_results = self.splunkService.jobs.oneshot(searchquery_oneshot, **kwargs_oneshot)

        sli = 0.0
        for result in results.JSONResultsReader(oneshotsearch_results):
            try:
                sli_value = float(result[metric])
            except ValueError as e:
                raise ValueError(f"failed to parse {metric}: {e}")
            sli = sli_value

        return sli

    def _get_metric_query(self, metric: str, start_time: int, end_time: int) -> str:
        query = self.custom_queries.get(metric, None)
        if query is not None:
            return self._replace_query_parameters(query, start_time, end_time)
        raise ValueError(f"No Custom query specified for metric {metric}")

    def _replace_query_parameters(self, query: str, start_time: str, end_time: str) -> str:
        
        query = query.replace("$PROJECT", self.project)
        query = query.replace("$STAGE", self.stage)
        query = query.replace("$SERVICE", self.service)

        # replace labels
        for key, value in self.labels.items():
            query = query.replace(f"$LABEL.{key}", value)

        # replace duration
        duration_seconds = str(
            math.floor((end_time - start_time)/100)
        ) + "s"

        query = query.replace("$DURATION_SECONDS", duration_seconds)
        return query

    def _parse_unix_timestamps(self, start_time: str, end_time: str) -> Tuple[int, int]:
        start_unix = int(datetime.fromisoformat(start_time).timestamp())
        end_unix = int(datetime.fromisoformat(end_time).timestamp())

        return start_unix, end_unix
    
if __name__=="__main__":
    sli = SplunkProvider(
        project='test-splunk',
        stage='qa',
        service='helloservice',
        labels={},
        customQueries={"test_query" : "search |inputcsv test.csv | stats count"}, 
        host='localhost', token='eyJraWQiOiJzcGx1bmsuc2VjcmV0IiwiYWxnIjoiSFM1MTIiLCJ2ZXIiOiJ2MiIsInR0eXAiOiJzdGF0aWMifQ.eyJpc3MiOiJhZG1pbiBmcm9tIGJhMzljNjk3ZTA5ZCIsInN1YiI6ImFkbWluIiwiYXVkIjoidGVzdCIsImlkcCI6IlNwbHVuayIsImp0aSI6IjU4MTRjNjBmNDNlNzk5ZDI1YzEzZDMyOWE4NTY2ZGM0ZmM5Mjg4MjQyMTg0NTAwMDY1NTdhYTYyYTI0YzYyNjQiLCJpYXQiOjE2Nzk0MTQxMjIsImV4cCI6MTY4MDQ1MDkyMiwibmJyIjoxNjc5NDE0MTIyfQ.gK2mdx7X8L6sdi50E0RvEI7wAvjEdq1P489pQ1isIRF5TzbL_RIXoB0Ku-zeRmo_Wc8hAcSNDPftu8QUuBCRkA',
        port=8000).get_sli('test_query', '2023-03-21T22:00:43.940','2023-03-21T22:02:50.940')
    print(sli)