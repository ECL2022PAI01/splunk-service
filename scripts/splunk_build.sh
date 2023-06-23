#!/bin/bash

get_splunk_pod() {
	echo $(kubectl -n keptn get pods | grep splunk | awk '{print $1}')
}
# go to the splunk service directory

if [[ $# -ne 3 ]] ; then
    echo 'please enter the splunk service directory and then the docker registry as arguments and then the app version'
    exit 1
fi
cd $1

# build a new docker image of the service

docker build . -t kuro08/splunk-service:$2 --network=host && docker push kuro08/splunk-service:$2

# remove an existing helm chart of the splunk service
if [[ $(get_splunk_pod)  ]] ; then
	helm uninstall -n keptn splunk-sli
	t=10
	kubectl -n keptn get pods
	echo "Waiting $t s for the previous splunk pods to be terminated"
	sleep $t
fi
# release the new chart
chartName=splunk-service-$2.tgz
tar -czvf $chartName chart/ && helm upgrade --install -n keptn splunk-service https://github.com/ECL2022PAI01/splunk-service/releases/download/$2/splunk-service-$2.tgz  --set splunkservice.existingSecret=splunk-service-secret

kubectl -n keptn get pods

t=20
echo "Waiting $t s for the splunk service pod to be running"
sleep $t
kubectl -n keptn logs -f -c splunk-service  $(get_splunk_pod)