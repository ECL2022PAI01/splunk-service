# splunk-service

![GitHub release (latest by date)](https://img.shields.io/github/v/release/ECL2022PAI01/splunk-service)
[![Go Report Card](https://goreportcard.com/badge/github.com/ECL2022PAI01/splunk-service)](https://goreportcard.com/report/github.com/ECL2022PAI01/splunk-service)
![GitHub release (latest by SemVer including pre-releases)](https://img.shields.io/github/downloads-pre/ECL2022PAI01/splunk-service/latest/total)

This implements the `splunk-service` that integrates the [splunk enterprise](https://en.wikipedia.org/wiki/splunk) platform with Keptn. This enables you to use splunk as the source for the Service Level Indicators ([SLIs](https://keptn.sh/docs/0.19.x/reference/files/sli/)) that are used for Keptn [Quality Gates](https://keptn.sh/docs/concepts/quality_gates/). The `splunk-service` can also use splunk for monitoring your microservice through saved searches alerts and triggering remediation sequences.
If you want to learn more about Keptn visit [keptn.sh](https://keptn.sh)

## Compatibility Matrix

Please always double-check the version of Keptn you are using compared to the version of this service, and follow the compatibility matrix below.

| Keptn Version | [splunk-service Docker Image](https://github.com/keptn-sandbox/splunk-service/pkgs/container/splunk-service) |
| :-----------: | :----------------------------------------------------------------------------------------------------------: |
|    0.18.x     |                                      ECL2022PAI01/splunk-service:0.3.x                                       |
|    0.19.x     |                                      ECL2022PAI01/splunk-service:0.3.x                                       |
|     1.x.y     |                                      ECL2022PAI01/splunk-service:0.3.x                                       |

## Installation instructions

### Install splunk

If you don't already have an instance of splunk running somewhere, you can install one via docker - you must have docker running on your machine ([how to install docker](https://docs.docker.com/engine/install)) -

Start an instance of a splunk enterprise ([see the docker page](https://hub.docker.com/r/splunk/splunk) for more details) :

```bash
docker pull splunk/splunk:latest

docker run -p 8089:<splunkdPort> -p 8000:<splunkInstancePort> -e "SPLUNK_START_ARGS=--accept-license" -e "SPLUNK_PASSWORD=mypassword" --name splunk-entreprise splunk/splunk:latest
```

### Install splunk-service

Please replace the placeholders in the commands below. Examples are provided.

- `<VERSION>`: splunk-service version, e.g., 0.1.0
- `<SPLUNK_HOST>` : where the splunk enterprise is installed, e.g, http://localhost
- `<SPLUNK_PORT>` : the port of the splunk enterprise server , e.g 8089
- `<SPLUNK_USERNAME>` : the username of the splunk server (**admin** by default)
- `<SPLUNK_PASSWORD>` : the password of the splunk server

_Note_: Make sure to replace `<VERSION>` with the version you want to install.

- Install Keptn splunk-service in Kubernetes using the following command. This will install the splunk-service into
  the `keptn` namespace.
  ```bash
  helm upgrade --install -n keptn splunk-service \
     https://github.com/ECL2022PAI01/splunk-service/releases/download/<VERSION>/splunk-service-<VERSION>.tgz \
     --set splunkservice.spHost="<SPLUNK_HOST>" \
     --set splunkservice.spPort=<SPLUNK_PORT>\
     --set splunkservice.spUsername="<SPLUNK_USERNAME>" \
     --set splunkservice.spPassword="<SPLUNK_PASSWORD>"
  ```
  This should install the `splunk-service` together with a Keptn `distributor` into the `keptn` namespace, which you can verify using
  ```console
  kubectl -n keptn get deployment splunk-service -o wide
  kubectl -n keptn get pods -l run=splunk-service
  ```
- (Optional) If you want to customize the namespaces of Keptn or the splunk installation, replace the environment
  variable values according to the use case:

  ```bash
  KEPTN_NAMESPACE="keptn"
  SPLUNK_NAMESPACE=<SPLUNK_NAMESPACE>

  helm upgrade --install -n ${KEPTN_NAMESPACE} splunk-service \
     https://github.com/ECL2022PAI01/splunk-service/releases/download/<VERSION>/splunk-service-<VERSION>.tgz \
     --set splunkservice.spHost="<SPLUNK_HOST>" \
     --set splunkservice.spPort=<SPLUNK_PORT>\
     --set splunkservice.spUsername="<SPLUNK_USERNAME>" \
     --set splunkservice.spPassword="<SPLUNK_PASSWORD>"
     --set splunk.service.namespace=${SPLUNK_NAMESPACE}
  ```

  - If you want to use an existing Secret in the cluster, make sure to have those environment variables : SP_HOST, SP_PORT and [SP_API_TOKEN, SP_SESSSION_KEY, {SP_USERNAME, SP_PASSWORD} ](token names should be an exact match) and run

  ```bash
  helm upgrade --install -n keptn splunk-service \
     https://github.com/ECL2022PAI01/splunk-service/releases/download/<VERSION>/splunk-service-<VERSION>.tgz \
     --set splunkservice.spHost="<SPLUNK_HOST>" \
     --set splunkservice.spPort=<SPLUNK_PORT>\
     --set splunkservice.existingSecret=<mysecretname>
  ```

- Execute the following command to configure splunk as the monitoring tool:

  ```bash
  keptn configure monitoring splunk --project=<PROJECT_NAME> --service=<SERVICE_NAME>
  ```

### Advanced Options

You can customize splunk-service with the following environment variables:
For splunk credentials :

```yaml
# Splunk host
- name: SP_HOST ""
  value: ""
# Splunk username if basic authentication is used
- name: SP_USERNAME ""
  value: "admin"
# Splunk password if basic authentication is used
- name: SP_PASSWORD ""
  value: ""
# Splunk token if authentication by token is used
- name: SP_API_TOKEN ""
  value: ""
# Splunk session if authentication by session key is used
- name: SP_SESSION_KEY ""
  value: ""
- name: SP_HOST ""
  value: ""
```

For customizing the alerts set when receiving a configure monitoring event :

```yaml
# The period during which the triggering of the alert is suppressed after it is already triggered. By default to "3m" (3 minutes)
- name: ALERT_SUPPRESS_PERIOD
  value: "{{ .Values.splunkservice.alertSuppressPeriod }}"
# A splunk expression specifying the frequency for the execution of the saved searches. By default to "*/1 * * * *" (every minute)
- name: CRON_SCHEDULE
  value: "{{ .Values.splunkservice.cronSchedule }}"
# The earliest time for the saved search. By default to "-3m"
- name: DISPATCH_EARLIEST_TIME
  value: "{{ .Values.splunkservice.dispatchEarliestTime }}"
# The latest time for the saved search. By default to "now"
- name: DISPATCH_LATEST_TIME
  value: "{{ .Values.splunkservice.dispatchLatestTime }}"
# The coma separated list of actions to perform after the triggering of alerts. By default to "". But can be "webhook"
- name: ACTIONS
  value: "{{ .Values.splunkservice.actions }}"
# Has to be set if webhook is one of the actions to be done
- name: WEBHOOK_URL
  value: "{{ .Values.splunkservice.webhookUrl }}"
```

#### Add SLI and SLO

Note that the sli.yaml should contain sli queries that are splunk searches returning each an atomic numeric value.

```bash
keptn add-resource --project="<your-project>" --stage="<stage-name>" --service="<service-name>" --resource=/path-to/your/sli-file.yaml --resourceUri=splunk/sli.yaml
keptn add-resource --project="<your-project>"  --stage="<stage-name>" --service="<service-name>" --resource=/path-to/your/slo-file.yaml --resourceUri=slo.yaml
```

Example:

```bash
keptn add-resource --project="podtatohead" --stage="hardening" --service="helloservice" --resource=./quickstart/sli.yaml --resourceUri=splunk/sli.yaml
keptn add-resource --project="podtatohead" --stage="hardening" --service="helloservice" --resource=./quickstart/slo.yaml --resourceUri=slo.yaml
```

### Configure Keptn to use splunk as SLI-provider

Use keptn CLI version [0.15.0](https://github.com/keptn/keptn/releases/tag/0.15.0) or later.

```bash
keptn configure monitoring splunk --project <project-name>  --service <service-name>
```

### Trigger delivery

```bash
keptn trigger delivery --project=<project-name> --service=<service-name> --image=<appRegistredToSplunk> --tag=<tag>
```

Example:

```bash
keptn trigger delivery --project=podtatohead --service=helloservice --image=docker.io/jetzlstorfer/helloserver --tag=0.1.1
```

Observe the results in the [Keptn Bridge](https://keptn.sh/docs/0.19.x/bridge/)

### Up- or Downgrading

Adapt and use the following command in case you want to up- or downgrade your installed version (specified by the `$VERSION` placeholder):

```console
helm upgrade -n keptn --set image.tag=$VERSION splunk-service chart/

# or via an official release
helm upgrade --install -n keptn splunk-service \
   https://github.com/ECL2022PAI01/splunk-service/releases/download/$VERSION/splunk-service-$VERSION.tgz \
   --reuse-values
```

### Uninstall

To delete a deployed _splunk-service_ helm chart:

```bash
helm uninstall splunk-service -n keptn
```

## Running tests on your local machine

Port-forward Keptn API so that our tests can create/delete Keptn resources

```bash
kubectl port-forward svc/api-gateway-nginx 8090:80 -n keptn # in a separate terminal window
```

from splunk-service repo

```bash
export ENABLE_E2E_TEST=true
```

```bash
export KEPTN_ENDPOINT=http://localhost:8090/api
```

```bash
export KEPTN_API_TOKEN=$(kubectl get secret keptn-api-token -n keptn -ojsonpath='{.data.keptn-api-token}' | base64 -d)
```

# Run tests

Unit tests

```bash
go test -v .
```

e2e test

```bash
gotestsum --format standard-verbose -- -timeout=120m  ./test/e2e/...
```
## How does it work?

### SLI Provider for quality gates

* The splunk-service allows keptn to use splunk as its SLI-provider for the quality gates. For an evaluation stage, when an sh.keptn.event.getsli.triggered is received by the splunk-service, that latter sends an sh.keptn.event.getsli.started, executes de splunk searches of the indicators and sends an sh.keptn.event.getsli.finished containing the results for the indicators.
In order for it to work properly, the slo.yaml and sli.yaml should be uploaded and the monitoring should be configured for the service as explained in the installation section.
* When the timeframe for the get-sli event is not specified in the splunk searches in sli.yaml (via "earliest" and "latest"), the default timeframe used for all the SLIs is the one specified in the shipyard.yaml or in the keptn bridge when only the evaluation is done. If "ealiest" and "latest" are specified in the splunk searches in sli.yaml, they will overwrite the default timeframe.
:warning: In case you use custom timeframes in your splunk searches, they will overwrite the timeframe from the shipyard or keptn bridge. The timeframe displayed in the keptn bridge is not correct. It's the timeframe from the shipyard or the keptn bridge that is displayed in the bridge.


### Monitoring and Remediation

* :warning: For using this functionality, AVOID using comas "," in the indicator names within the sli.yaml file.
* The splunk-service allows keptn to use splunk in order to monitor the deployed service. Executing the command "keptn configure monitoring splunk --project=<project> --service=<service>" sends an sh.keptn.configure-monitoring.triggered event. Whenever the splunk-service receives that event, it sends the corresponding .started event, creates splunk alerts from the SLIs and SLOs for the stages where slo.yaml and remediation.yaml files are defined and finally sends the corresponding .finished event to keptn. The splunk alerts created are saved searches that run in a periodic way and are in a fired state whenever the alert conditions are met. See the advanced options section for more information.
* The splunk-service checks periodically whether or not one of the keptn splunk alerts is triggered. Once it detects a triggered keptn alert, an sh.keptn.event.remediation.triggered event is sent to keptn with the details concerning the problem. Keptn then executes the remediation actions specified in the remediation file. 
* Splunk alerts are deleted and recreated for a particular service in a particular project whenever the keptn configure monitoring command is executed for splunk. This way, it is possible to UPDATE the splunk alerts when changes have been made regarding the sli.yaml and slo.yaml.
* If you only want to DELETE the keptn splunk alerts concerning a particular service in a particular project without updating them, just delete one of these : the remediation file, the sli file, the slo file, the service OR the entire project and then execute :
```bash
keptn configure monitoring splunk --project <project>  --service <service>
```


## Development


Development can be conducted using any GoLang compatible IDE/editor (e.g., Jetbrains GoLand, VSCode with Go plugins).

It is recommended to make use of branches as follows:

- `master` contains the latest potentially unstable version
- `release-*` contains a stable version of the service (e.g., `release-0.1.0` contains version 0.1.0)
- create a new branch for any changes that you are working on, e.g., `feature/my-cool-stuff` or `bug/overflow`
- once ready, create a pull request from that branch back to the `master` branch

When writing code, it is recommended to follow the coding style suggested by the [Golang community](https://github.com/golang/go/wiki/CodeReviewComments).

### Where to start

If you don't care about the details, your first entrypoint is [handler](eventhandlers.go). Within this folder
you can add implementation for pre-defined Keptn Cloud events.

To better understand all variants of Keptn CloudEvents, please look at the [Keptn Spec](https://github.com/keptn/spec).

If you want to get more insights into processing those CloudEvents or even defining your own CloudEvents in code, please
look into [main.go](main.go) (specifically `processKeptnCloudEvent`), [chart/templates](chart/templates),
consult the [Keptn docs](https://keptn.sh/docs/) as well as existing [Keptn Core](https://github.com/keptn/keptn) and
[Keptn Contrib](https://github.com/keptn-contrib/) services.

### Tips

If you want to build and deploy a new version of your updated service, there is a script (located in [scripts/build-splunk-service.sh](scripts/build-splunk-service.sh)) that will build and deploy the service to your cluster. 

```bash
# It takes the following arguments:
# -d: the path to the splunk service directory
# -u: the docker registry username
# -v: the version of the service to build and push
# -l: show the logs of the splunk service pod
./scripts/build-splunk-service.sh -d ../. -u dockerUsername -v 0.1.0 -l
```

### Common tasks

- Build the binary: `go build -ldflags '-linkmode=external' -v -o splunk-service`
- Run tests: `go test -race -v ./...`
- Build the docker image: `docker build . -t ghcr.io/keptn-sandbox/splunk-service:latest`
- Run the docker image locally: `docker run --rm -it -p 8080:8080 ghcr.io/keptn-sandbox/splunk-service:latest`
- Push the docker image to DockerHub: `docker push ghcr.io/keptn-sandbox/splunk-service:latest`
- Watch the deployment using `kubectl`: `kubectl -n keptn get deployment splunk-service -o wide`
- Get logs using `kubectl`: `kubectl -n keptn logs deployment/splunk-service -f`
- Watch the deployed pods using `kubectl`: `kubectl -n keptn get pods -l run=splunk-service`

### Testing Cloud Events

We have dummy cloud-events in the form of [RFC 2616](https://ietf.org/rfc/rfc2616.txt) requests in the [test-events/](test-events/) directory. These can be easily executed using third party plugins such as the [Huachao Mao REST Client in VS Code](https://marketplace.visualstudio.com/items?itemName=humao.rest-client).

## How to release a new version of this service

It is assumed that the current development takes place in the master branch (either via Pull Requests or directly).

To make use of the built-in automation using GH Actions for releasing a new version of this service, you should

- branch away from master to a branch called `release-x.y.z` (where `x.y.z` is your version),
- check the output of GH Actions builds for the release branch,
- verify that your image was built and pushed to GHCR with the right tags,
- update the image tags in [deploy/service.yaml], and
- test your service against a working Keptn installation.

If any problems occur, fix them in the release branch and test them again.

Once you have confirmed that everything works and your version is ready to go, you should

- create a new release on the release branch using the [GitHub releases page](https://github.com/keptn-sandbox/splunk-service/releases), and
- merge any changes from the release branch back to the master branch.

## Known problems

## License

Please find more information in the [LICENSE](LICENSE) file.
