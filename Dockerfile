FROM golang:1.20 AS build

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

RUN go test ./... && \
    go build -v -o circleci-runner-autoscaler

FROM debian:buster-slim

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/circleci-runner-autoscaler /app/circleci-runner-autoscaler
CMD ["/app/circleci-runner-autoscaler"]
