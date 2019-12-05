# PoW: Blockchain Proof-of-Work in the Cloud
![](https://travis-ci.com/jaylees14/pow.svg?token=DHJ1zWJnxL4gE1gKLsuC&branch=master)

This project was created for the COMSM0010 Cloud Computing Assignment: Horizontal Scaling.
It demonstrates how horizontal-scaling cloud-based infrastructure can be leveraged to parallelise the "proof of work" stage of the blockchain distributed ledger protocol.

Features include:
- Direct and indirect machine specification
- Task deployment on AWS infrastructure using either Docker or ECS
- Grafana dashboards, pulling metrics from Prometheus
- cAdvisor metrics per container, as well as custom worker metrics using Prometheus SDK

## Project Structure
The project is separated into 3 separate components:

- `/client`: houses the local client based code which deploys and controls the cloud infrastructure
- `/grafana`: houses the custom Grafana dashboards and associated Dockerfile
- `/worker`: houses the worker scripts to compute the "Golden Nonce", and associated Dockerfile

## Usage
- Firstly, administrator AWS credentials must be present in the file `~/.aws/credentials`
- Then, install Go as per the instructions [here](https://golang.org/doc/install)
- Run `go get -u github.com/aws/aws-sdk-go/...`
- From the `client` directory, run the following command for a list of available options

```
~/g/s/g/j/p/client ❯❯❯ go run main.go -help
[direct] mode
  -block string
        block of data the nonce is appended to (default "COMSM0010cloud")
  -d int
        number of leading zeros (default 20)
  -n int
        number of workers (default 1)
  -timeout int
        timeout in seconds (default 360)
  -use-ecs
        use ecs as a task scheduler

[indirect] mode
  -block string
        block of data the nonce is appended to (default "COMSM0010cloud")
  -confidence int
        confidence in finding the result, as a percentage (default 95)
  -d int
        number of leading zeros (default 20)
  -timeout int
        timeout in seconds (default 360)
  -use-ecs
        use ecs as a task scheduler
```

## Deploying Containers
Each of the containers, Grafana and Worker, are deployed on Docker Hub.
Travis CI is configured to deploy these upon every push, however this can be manually triggered by executing the `deploy.sh` script.
