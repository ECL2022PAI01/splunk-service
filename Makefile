IMG_TAG ?= dev
IMG_REPO ?= kuro08/splunk-service

docker-build:
	docker build . -t $(IMG_REPO):$(IMG_TAG)

docker-run:
	docker run --rm -it -p 8080:8080  $(IMG_REPO):$(IMG_TAG)

docker-push:
	docker push $(IMG_REPO):$(IMG_TAG)