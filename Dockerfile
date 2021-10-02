
####--------------
FROM golang:1.14.13-alpine3.12  AS build-env
RUN echo $GOPATH

RUN apk add --no-cache git gcc musl-dev
RUN apk add --update make

WORKDIR /go/src/github.com/devtron-labs/cirunner
ADD . /go/src/github.com/devtron-labs/cirunner/
COPY . .
RUN pwd
RUN echo $GOPATH
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/cirunner


FROM docker:18.09.7-dind
# All these steps will be cached
#RUN apk add --no-cache ca-certificates
RUN apk update
RUN apk add --no-cache --virtual .build-deps
RUN apk add bash
RUN apk add make && apk add curl && apk add openssh
RUN apk add git
RUN apk add zip
RUN ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime
RUN apk -Uuv add groff less python py-pip
RUN pip install awscli
RUN apk --purge -v del py-pip
RUN rm /var/cache/apk/*
COPY --from=docker/compose:latest /usr/local/bin/docker-compose /usr/bin/docker-compose

COPY ./git-ask-pass.sh /git-ask-pass.sh
RUN chmod +x /git-ask-pass.sh

COPY --from=build-env /go/bin/cirunner .
COPY ./ssh-config /root/.ssh/config
ENTRYPOINT ["./cirunner"]