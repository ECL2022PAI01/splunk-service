IMG_TAG ?= latest
IMG_REPO ?= Mouhamadou305/splunk-service

docker-build:
	docker build . -t $(IMG_REPO):$(IMG_TAG)
	
docker-push:
	docker push $(IMG_REPO):$(IMG_TAG)