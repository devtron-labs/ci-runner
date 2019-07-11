
####--------------
FROM golang:1.12.6-alpine3.9  AS build-env
RUN echo $GOPATH

RUN apk add --no-cache git gcc musl-dev
RUN apk add --update make
RUN mkdir /cirunner
WORKDIR /cirunner
COPY . .
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/cirunner


FROM docker:18.09.7-dind
# All these steps will be cached
RUN apk update
RUN apk add --no-cache --virtual .build-deps
RUN apk add bash
RUN apk add make && apk add curl && apk add openssh
RUN apk add git
RUN ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime
RUN apk -Uuv add groff less python py-pip
RUN pip install awscli
RUN apk --purge -v del py-pip
RUN rm /var/cache/apk/*
COPY --from=build-env /go/bin/cirunner .
ENTRYPOINT ["./cirunner"]


