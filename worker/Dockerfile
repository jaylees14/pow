FROM golang:1.8.3-alpine

WORKDIR /go/src/github.com/jaylees14/pow/worker

COPY . .

RUN apk add --no-cache git
RUN go get github.com/aws/aws-sdk-go/aws

RUN go build -o client

ENTRYPOINT ./client
