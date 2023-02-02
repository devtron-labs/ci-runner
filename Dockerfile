FROM golang:1.17.8-alpine3.15  AS build-env

RUN apk add --no-cache git gcc musl-dev
RUN apk add --update make

WORKDIR /go/src/github.com/devtron-labs/cirunner
ADD . /go/src/github.com/devtron-labs/cirunner/
COPY . .
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/cirunner

FROM quay.io/podman/stable:v4.3.1
# All these steps will be cached

RUN dnf upgrade && dnf install bash make curl openssh git zip jq -y -q
RUN ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime
RUN dnf -v install groff less python3 python3-pip -y -q
RUN pip3 install awscli
RUN dnf remove python3-pip -y
COPY --from=docker/compose:latest /usr/local/bin/docker-compose /usr/bin/docker-compose

COPY ./podman/registeries.conf /etc/containers/registries.conf
COPY ./buildpack.json /buildpack.json
COPY ./git-ask-pass.sh /git-ask-pass.sh
RUN chmod +x /git-ask-pass.sh
RUN rm -rf ./.docker

RUN cd /usr/bin \
    && ln -s podman docker

RUN (curl -sSL "https://github.com/buildpacks/pack/releases/download/v0.27.0/pack-v0.27.0-linux.tgz" | tar -C /usr/local/bin/ --no-same-owner -xzv pack)

COPY --from=build-env /go/bin/cirunner .
COPY ./ssh-config /root/.ssh/config
ENTRYPOINT ["./cirunner"]
