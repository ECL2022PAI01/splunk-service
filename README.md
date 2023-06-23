- [splunk-service](#splunk-service)
  * [Quickstart](#quickstart)
  * [If you already have a Keptn cluster running](#if-you-already-have-a-keptn-cluster-running)
  * [Compatibility Matrix](#compatibility-matrix)
  * [Installation](#installation)
    + [Up- or Downgrading](#up--or-downgrading)
    + [Uninstall](#uninstall)
  * [Running tests on your local machine](#running-tests-on-your-local-machine)
    + [Run tests](#run-tests)
  * [Development](#development)
    + [Where to start](#where-to-start)
    + [Common tasks](#common-tasks)
    + [Testing Cloud Events](#testing-cloud-events)
  * [Automation](#automation)
    + [GitHub Actions: Automated Pull Request Review](#github-actions-automated-pull-request-review)
    + [GitHub Actions: Unit Tests](#github-actions-unit-tests)
  * [How to release a new version of this service](#how-to-release-a-new-version-of-this-service)
  * [Known problems](#known-problems)
  * [License](#license)
# splunk-service
![GitHub release (latest by date)]()
[![Go Report Card]()

This implements the `splunk-service` that integrates the [splunk](https://en.wikipedia.org/wiki/splunk) platform with Keptn. This enables you to use splunk as the source for the Service Level Indicators ([SLIs](https://keptn.sh/docs/0.19.x/reference/files/sli/)) that are used for Keptn [Quality Gates](https://keptn.sh/docs/concepts/quality_gates/).
If you want to learn more about Keptn visit [keptn.sh](https://keptn.sh)


## Quickstart
If you are on Mac or Linux, you can use [examples/kup.sh](./examples/kup.sh) to set up a local Keptn installation that uses splunk. This script creates a local minikube cluster, installs Keptn, splunk and the splunk integration for Keptn (check the script for pre-requisites). 

To use the script,
```bash
examples/kup.sh
```

## If you already have a Keptn cluster running
1. Install splunk

Add splunk helm repo:
```bash
helm repo add splunk https://helm.splunkhq.com
```
Install splunk helm chart:
```bash
helm install splunk --set splunk.apiKey=${DD_API_KEY} splunk/splunk --set splunk.appKey=${DD_APP_KEY} --set splunk.site=${DD_SITE} --set clusterAgent.enabled=true --set clusterAgent.metricsProvider.enabled=true --set clusterAgent.createPodDisruptionBudget=true --set clusterAgent.replicas=2

```
2. Install Keptn splunk-service to integrate splunk with Keptn
```bash
# cd splunk-service
helm install splunk-service ./helm
```

3. Add SLI and SLO
```bash
keptn add-resource --project="<your-project>" --stage="<stage-name>" --service="<service-name>" --resource=/path-to/your/sli-file.yaml --resourceUri=splunk/sli.yaml
keptn add-resource --project="<your-project>"  --stage="<stage-name>" --service="<service-name>" --resource=/path-to/your/slo-file.yaml --resourceUri=slo.yaml
```
Example:
```bash
keptn add-resource --project="podtatohead" --stage="hardening" --service="helloservice" --resource=./quickstart/sli.yaml --resourceUri=splunk/sli.yaml
keptn add-resource --project="podtatohead" --stage="hardening" --service="helloservice" --resource=./quickstart/slo.yaml --resourceUri=slo.yaml
```
Check [./quickstart/sli.yaml](./examples/quickstart/sli.yaml) and [./quickstart/slo.yaml](./examples/quickstart/slo.yaml) for example SLI and SLO. 

4. Configure Keptn to use splunk SLI provider
Use keptn CLI version [0.15.0](https://github.com/keptn/keptn/releases/tag/0.15.0) or later.
```bash
keptn configure monitoring splunk --project <project-name>  --service <service-name>
```

5. Trigger delivery
```bash
keptn trigger delivery --project=<project-name> --service=<service-name> --image=<image> --tag=<tag>
```
Example:
```bash
keptn trigger delivery --project=podtatohead --service=helloservice --image=docker.io/jetzlstorfer/helloserver --tag=0.1.1
```
Observe the results in the [Keptn Bridge](https://keptn.sh/docs/0.19.x/bridge/)
## Compatibility Matrix

*Please fill in your versions accordingly*

| Keptn Version    | [splunk-service Docker Image](https://github.com/keptn-sandbox/splunk-service/pkgs/container/splunk-service) |
|:----------------:|:----------------------------------------:|
|       x.y.z       | ghcr.io/keptn-sandbox/splunk-service:x.y.z |
|       u.v.w      | ghcr.io/keptn-sandbox/splunk-service:u.v.w |
 

## Installation

```bash
# cd splunk-service
helm install splunk-service ./helm --set splunkservice.ddApikey=${DD_API_KEY} --set splunkservice.ddAppKey=${DD_APP_KEY} --set splunkservice.ddSite=${DD_SITE}
```
Tell Keptn to use splunk as SLI provider for your project/service
```bash
keptn configure monitoring splunk --project <project-name>  --service <service-name>
```

This should install the `splunk-service` together with a Keptn `distributor` into the `keptn` namespace, which you can verify using

```console
kubectl -n keptn get deployment splunk-service -o wide
kubectl -n keptn get pods -l run=splunk-service
```
### Up- or Downgrading

Adapt and use the following command in case you want to up- or downgrade your installed version (specified by the `$VERSION` placeholder):

```bash
helm upgrade splunk-service ./helm
```

### Uninstall

To delete a deployed *splunk-service* helm chart:

```bash
helm uninstall splunk-service
```
## Running tests on your local machine
port-forward Keptn API so that our tests can create/delete Keptn resources

``` bash
kubectl port-forward svc/api-gateway-nginx 5000:80 -nkeptn # in a separate terminal window
``` 

from splunk-service repo

```bash 
export ENABLE_E2E_TEST=true
```

```bash
export KEPTN_ENDPOINT=http://localhost:5000/api
```

```bash 
export KEPTN_API_TOKEN=$(kubectl get secret keptn-api-token -n keptn -ojsonpath='{.data.keptn-api-token}' | base64 -d)
```


# Run tests
```bash
gotestsum --format standard-verbose -- -timeout=120m  ./test/e2e/...
```

## Development

Development can be conducted using any GoLang compatible IDE/editor (e.g., Jetbrains GoLand, VSCode with Go plugins).

It is recommended to make use of branches as follows:

* `master` contains the latest potentially unstable version
* `release-*` contains a stable version of the service (e.g., `release-0.1.0` contains version 0.1.0)
* create a new branch for any changes that you are working on, e.g., `feature/my-cool-stuff` or `bug/overflow`
* once ready, create a pull request from that branch back to the `master` branch

When writing code, it is recommended to follow the coding style suggested by the [Golang community](https://github.com/golang/go/wiki/CodeReviewComments).

### Where to start

If you don't care about the details, your first entrypoint is [eventhandlers.go](eventhandlers.go). Within this file 
 you can add implementation for pre-defined Keptn Cloud events.
 
To better understand all variants of Keptn CloudEvents, please look at the [Keptn Spec](https://github.com/keptn/spec).
 
If you want to get more insights into processing those CloudEvents or even defining your own CloudEvents in code, please 
 look into [main.go](main.go) (specifically `processKeptnCloudEvent`), [helm/templates](helm/templates),
 consult the [Keptn docs](https://keptn.sh/docs/) as well as existing [Keptn Core](https://github.com/keptn/keptn) and
 [Keptn Contrib](https://github.com/keptn-contrib/) services.

### Common tasks

* Build the binary: `go build -ldflags '-linkmode=external' -v -o splunk-service`
* Run tests: `go test -race -v ./...`
* Build the docker image: `docker build . -t ghcr.io/keptn-sandbox/splunk-service:latest`
* Run the docker image locally: `docker run --rm -it -p 8080:8080 ghcr.io/keptn-sandbox/splunk-service:latest`
* Push the docker image to DockerHub: `docker push ghcr.io/keptn-sandbox/splunk-service:latest`
* Watch the deployment using `kubectl`: `kubectl -n keptn get deployment splunk-service -o wide`
* Get logs using `kubectl`: `kubectl -n keptn logs deployment/splunk-service -f`
* Watch the deployed pods using `kubectl`: `kubectl -n keptn get pods -l run=splunk-service`


### Testing Cloud Events

We have dummy cloud-events in the form of [RFC 2616](https://ietf.org/rfc/rfc2616.txt) requests in the [test-events/](test-events/) directory. These can be easily executed using third party plugins such as the [Huachao Mao REST Client in VS Code](https://marketplace.visualstudio.com/items?itemName=humao.rest-client).

## Automation

### GitHub Actions: Automated Pull Request Review

[comment]: # (This repo uses [reviewdog](https://github.com/reviewdog/reviewdog) for automated reviews of Pull Requests.)

[comment]: # (You can find the details in [.github/workflows/reviewdog.yml](.github/workflows/reviewdog.yml).)

### GitHub Actions: Unit Tests

[comment]: # (This repo has automated unit tests for pull requests. )

[comment]: # (You can find the details in [.github/workflows/tests.yml](.github/workflows/tests.yml).)

## How to release a new version of this service

It is assumed that the current development takes place in the master branch (either via Pull Requests or directly).

To make use of the built-in automation using GH Actions for releasing a new version of this service, you should

* branch away from master to a branch called `release-x.y.z` (where `x.y.z` is your version),
* check the output of GH Actions builds for the release branch, 
* verify that your image was built and pushed to GHCR with the right tags,
* update the image tags in [deploy/service.yaml], and
* test your service against a working Keptn installation.

If any problems occur, fix them in the release branch and test them again.

Once you have confirmed that everything works and your version is ready to go, you should

* create a new release on the release branch using the [GitHub releases page](https://github.com/keptn-sandbox/splunk-service/releases), and
* merge any changes from the release branch back to the master branch.

## Known problems
1. Does not support default queries for throughput, error rate, request latency etc., i.e., you have to enter the entire query. 

## License

Please find more information in the [LICENSE](LICENSE) file.
