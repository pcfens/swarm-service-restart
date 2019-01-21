FROM golang:1.11-alpine as build

WORKDIR /go/src/github.com/pcfens/swarm-service-restart
COPY . /go/src/github.com/pcfens/swarm-service-restart

RUN CGO_ENABLED=0 GOOS=linux go build -a -o swarm-service-restart

FROM scratch

ENV DOCKER_API_VERSION='1.30'
COPY --from=build /go/src/github.com/pcfens/swarm-service-restart/swarm-service-restart swarm-service-restart

ENTRYPOINT ["/swarm-service-restart"]
