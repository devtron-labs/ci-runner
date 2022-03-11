
####--------------
FROM golang:1.17.8-alpine3.12  AS build-env

RUN apk add --no-cache git gcc musl-dev
RUN apk add --update make

WORKDIR /go/src/github.com/devtron-labs/cirunner
ADD . /go/src/github.com/devtron-labs/cirunner/
COPY . .
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/cirunner


FROM docker:20.10.12-dind
# All these steps will be cached
#RUN apk add --no-cache ca-certificates
RUN apk update && add --no-cache --virtual .build-deps && add bash && add make && apk add curl && apk add openssh && add git && add zip
RUN ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime
RUN apk -Uuv add groff less python3 py3-pip
RUN pip3 install awscli
RUN apk --purge -v del py-pip
RUN rm /var/cache/apk/*
COPY --from=docker/compose:latest /usr/local/bin/docker-compose /usr/bin/docker-compose

COPY ./git-ask-pass.sh /git-ask-pass.sh
RUN chmod +x /git-ask-pass.sh

COPY --from=build-env /go/bin/cirunner .
COPY ./ssh-config /root/.ssh/config
ENTRYPOINT ["./cirunner"]
