FROM golang

WORKDIR /go/src/github.com/jaylees14/pow/worker

COPY . .

RUN go get github.com/aws/aws-sdk-go/aws
RUN go get github.com/prometheus/client_golang/prometheus
RUN go get github.com/prometheus/client_golang/prometheus/promauto
RUN go get github.com/prometheus/client_golang/prometheus/promhttp

RUN go build -o client

EXPOSE 2112
ENTRYPOINT ./client
