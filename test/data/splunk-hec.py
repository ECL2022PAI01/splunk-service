import requests, os, time, re, json
from datetime import datetime
from random import randint
from dotenv import load_dotenv
from logging import log
import logging


class Splunksender(object):
    def __init__(self, instance, token):
        self.instance = instance
        self.token = token

    def send(self, doc, source):
        headers = {"Authorization": f"Splunk {self.token}"}
        body = {"event": doc}
        body["index"] = "keptn-splunk-dev"
        body["sourcetype"] = "httpevent"
        body["source"] = source
        response = requests.request(
            "POST",
            f"https://{self.instance}",
            headers=headers,
            data=json.dumps(body),
            verify=False,
        )
        return response


def updateDate(log_line: str) -> str:
    current_date = f"[{datetime.now().strftime('%a %b %d %H:%M:%S %Y')}]"
    new_line = re.sub(r"\[.*?\]", current_date, log_line, count=1)

    return new_line


if __name__ == "__main__":
    load_dotenv()
    logging.basicConfig(level=logging.INFO)

    host = os.getenv("SPLUNK_HEC_HOST")
    port = os.getenv("SPLUNK_HEC_PORT")
    token = os.getenv("SPLUNK_HEC_TOKEN")
    fileName = os.getenv("SPLUNK_LOG_FILE_NAME")

    if not host or not port or not token or not fileName:
        raise EnvironmentError(
            "Please set the environment variables SPLUNK_HEC_HOST, SPLUNK_HEC_PORT and SPLUNK_HEC_TOKEN and SPLUNK_LOG_FILE_NAME"
        )
    instance = f"{host}:{port}/services/collector"

    log(level=logging.INFO, msg=f"Sending to {instance}")
    sp = Splunksender(instance, token)

    # read the log files first and update the date to the current date

    with open(fileName, "r") as file:
        log(level=logging.INFO, msg="Reading file")
        data = file.readlines()
        log(level=logging.INFO, msg=len(data))
        for d in data:
            resp = sp.send(updateDate(d), "http:podtato-error")
            log(level=logging.INFO, msg=d)
            log(level=logging.INFO, msg=f"Response: {resp.content}")
            time.sleep(randint(1, 2))
    # write the updated log lines to the file
    # with open(fileName, "w") as file:
    #     file.writelines(tmpFile)

    resp.close()
