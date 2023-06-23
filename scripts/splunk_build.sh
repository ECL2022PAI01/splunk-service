#!/bin/bash

# go to the splunk service directory

if [[ $# -ne 3 ]] ; then
    echo 'please enter the splunk service directory and then the docker registry as arguments and then the app version'
    exit 1
fi
cd $1  

# build a new docker image of the service

docker build . -t $2 --network=host && docker push $2:latest

# remove an existing helm chart of the splunk service
helm uninstall -n keptn splunk-sli
t=10
kubectl -n keptn get pods
echo "Waiting $t s for the previous splunk pods to be terminated"
sleep $t

# release the new chart
tar -czvf test/splunk/splunkChart.tgz chart/ && helm upgrade --install -n keptn splunk-sli test/splunk/splunkChart.tgz --set splunkservice.existingSecret=splunk-service-secret


kubectl -n keptn get pods

pod=$(kubectl -n keptn get pods | grep splunk | awk '{print $1}')
t=15
echo "Waiting $t s for the splunk service pod to be running"
sleep $t
kubectl -n keptn logs -f -c splunk-service $pod