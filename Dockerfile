FROM golang:1.11-alpine as build

WORKDIR /go/src/github.com/pcfens/swarm-service-restart
COPY . /go/src/github.com/pcfens/swarm-service-restart

RUN CGO_ENABLED=0 GOOS=linux go build -a -o swarm-service-restart

FROM scratch

COPY --from=build /go/src/github.com/pcfens/swarm-service-restart swarm-service-restart

ENTRYPOINT ["/swarm-service-restart"]
