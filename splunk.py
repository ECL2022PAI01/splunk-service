from logging import error
import logging
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
        customQueries= "",
        host: str="",
        username: str="",
        password: str="",
        token: str="",
        autologin= True,
        port: str="8089"
    ) -> None:
        if(username!="" and password!=""):
            #connecting using username and password
            self.splunkService = client.connect(host=host, port=int(port), username=username, password=password, autologin=autologin, scheme="https", basic=True)
        elif(token!="" and host!=""):
            #connecting using bearer token
            self.splunkService = client.connect(host=host, port=int(port), splunkToken=token, autologin=autologin, scheme="https", basic=True)
        else:
            logging.info("Connection credentials are invalid")
            
        self.splunkService._instance_type = "cloud"
        self.splunkService._splunk_version = (8, 2, 0)
        self.project = project
        self.stage = stage
        self.service= service
        self.labels = labels
        self.custom_queries = customQueries
        logging.info("ATTENTION PYTHON CUSTOMQUERRY")
        logging.info(customQueries)

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
                # raise ValueError(f" HELLLLP {result}")
                sli_value = float(result[list(result)[0]])
            except ValueError as e:
                raise ValueError.info(f"failed to parse {metric}: {e} with result : {result}")
            sli = sli_value

        return sli

    def _get_metric_query(self, metric: str, start_time: int, end_time: int) -> str:
        logging.info("ATTENTION PYTHON METRIC")
        logging.info(self.custom_queries.get(metric, None))
        query = self.custom_queries.get(metric, None)
        if query is not None:
            return self._replace_query_parameters(query, start_time, end_time)
        raise ValueError.info(f"No Custom query specified for metric {metric}")

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
    
# if __name__=="__main__":
#     sli= SplunkProvider(project='test',stage='qa',service='helloservice',labels={}, customQueries={'test_query' : '|inputcsv test.csv | stats count'}, host='20.74.14.146', token='eyJraWQiOiJzcGx1bmsuc2VjcmV0IiwiYWxnIjoiSFM1MTIiLCJ2ZXIiOiJ2MiIsInR0eXAiOiJzdGF0aWMifQ.eyJpc3MiOiJhZG1pbiBmcm9tIHNwbHVuay1lbnRyZXByaXNlLWRlcGxveW1lbnQtNzU2NjU0ZDhjLXptOWpiIiwic3ViIjoiYWRtaW4iLCJhdWQiOiJrZXB0biIsImlkcCI6IlNwbHVuayIsImp0aSI6IjU5MzRhN2VhYmYzNjIxOGY1ODQyYjYyYWJlZjZjNjIyNGFhNDM1ODA0ZTFlNmE3NGMxMjJjZGMwM2U0YTdmOTgiLCJpYXQiOjE2ODI1NDAzMDQsImV4cCI6MTY4NTEzMjMwNCwibmJyIjoxNjgyNTQwMzA0fQ.Xe9mI_CHGC_LYS80HlbUPbX9SpxidqukZACPZrgFPptx6LiCoRhtYJtgXlJMc3mXzJbYwdDZmK3TWypdemPsfA', port='8089').get_sli('test_query', '2023-04-03T04:40:03.880','2023-04-03T04:42:03.880')
#     print(sli)
    
