
all: build

TAG?=6bd516eacd6ee-10
FLAGS=
ENVVAR=
GOOS?=darwin
REGISTRY?=686244538589.dkr.ecr.us-east-2.amazonaws.com
BASEIMAGE?=alpine:3.9
#BUILD_NUMBER=$$(date +'%Y%m%d-%H%M%S')
#BUILD_NUMBER := $(shell bash -c 'echo $$(date +'%Y%m%d-%H%M%S')')
include $(ENV_FILE)
export

build: clean
	$(ENVVAR) GOOS=$(GOOS) go build -o cirunner

clean:
	rm -f cirunner

run: build
	./cirunner

.PHONY: build
docker-build-image:  build
	 docker build -t testci-runner:$(TAG) .

.PHONY: build, all, wire, clean, run, set-docker-build-env, docker-build-push, orchestrator,
docker-build-push: docker-build-image
	docker tag testci-runner:${TAG}  ${REGISTRY}/testci-runner:${TAG}
	docker push ${REGISTRY}/testci-runner:${TAG}



