FROM docker-registry:8080/centos-golang:latest as test
ARG VERSION=latest
COPY . /go/src/gitlab.trading.imc.intra/network/flower/
RUN cd /go/src/gitlab.trading.imc.intra/network/flower/ ;\
    go fmt ./... && go get -d -t ./...;\
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o flower .


FROM docker-registry:8080/alpine

COPY --from=0 /go/src/gitlab.trading.imc.intra/network/flower/flower /usr/src/app/flower
COPY --from=0 /go/src/gitlab.trading.imc.intra/network/flower/config /usr/src/app/config
COPY --from=0 /go/src/gitlab.trading.imc.intra/network/flower/templates /usr/src/app/templates
WORKDIR /usr/src/app/

ENTRYPOINT ["/usr/src/app/flower"]
