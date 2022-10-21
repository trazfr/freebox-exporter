FROM golang:1.19 as build-env

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN go build -o /go/bin/freebox-exporter

#####

FROM gcr.io/distroless/base

LABEL maintainer="" \
    org.opencontainers.image.authors="The Freebox Prometheus Exporter Authors" \
    org.opencontainers.image.title="gcr.io/nlamirault/freebox-exporter" \
	org.opencontainers.image.description="A Prometheus exporter for the Freebox, a Set-Top-Box (TV box) provided by French Internet Service Provider Bouygues Telecom" \
	org.opencontainers.image.documentations="https://github.com/nlamirault/freebox-exporter" \
    org.opencontainers.image.url="https://github.com/nlamirault/freebox-exporter" \
	org.opencontainers.image.source="git@github.com:nlamirault/freebox-exporter" \
    org.opencontainers.image.licenses="Apache 2.0" \
    org.opencontainers.image.vendor=""

COPY --from=build-env /go/bin/freebox-exporter /
# set the uid as an integer for compatibility with runAsNonRoot in Kubernetes
USER 65534:65534
CMD ["/freebox-exporter"]
