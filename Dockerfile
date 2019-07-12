
####--------------
FROM golang:1.12.6-alpine3.9  AS build-env
RUN echo $GOPATH

RUN apk add --no-cache git gcc musl-dev
RUN apk add --update make

WORKDIR /go/src/devtron.ai/cirunner
ADD . /go/src/devtron.ai/cirunner/
COPY . .
RUN pwd
RUN echo $GOPATH
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/cirunner


FROM docker:18.09.7-dind
# All these steps will be cached
RUN apk add --no-cache ca-certificates
COPY --from=build-env /go/bin/cirunner .
ENTRYPOINT ["./cirunner"]


