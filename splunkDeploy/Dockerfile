FROM  splunk/splunk:latest

ENV SPLUNK_START_ARGS=--accept-license
ENV SPLUNK_PASSWORD="mypassword"

COPY test.csv /opt/splunk/var/run/splunk/csv/test.csv
EXPOSE 8000

CMD ["start"]