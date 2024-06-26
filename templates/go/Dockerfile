FROM --platform=${TARGETPLATFORM:-linux/amd64} ghcr.io/openfaas/of-watchdog:0.9.15 as watchdog
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.22-alpine as build

RUN go install -v github.com/incu6us/goimports-reviser/v3@latest
RUN go install golang.org/x/tools/cmd/goimports@latest

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN apk --no-cache add git

COPY --from=watchdog /fwatchdog /usr/bin/fwatchdog
RUN chmod +x /usr/bin/fwatchdog


RUN mkdir -p /go/src/handler
WORKDIR /go/src/handler

COPY . .


ARG GO111MODULE="on"
ARG GOPROXY=""
ARG GOFLAGS=""
ARG CGO_ENABLED=0
ENV CGO_ENABLED=${CGO_ENABLED}

RUN goimports -w .
RUN goimports-reviser -format -recursive -rm-unused ./...
RUN go mod download


WORKDIR /go/src/handler
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build --ldflags "-s -w" -o handler.bin .


## Final image
FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:3.18.4 as ship

# Add non root user and certs
RUN apk --no-cache add ca-certificates \
    && addgroup -S app && adduser -S -g app app

# Split instructions so that buildkit can run & cache
# the previous command ahead of time.
RUN mkdir -p /home/app \
    && chown app /home/app

WORKDIR /home/app

COPY --from=build --chown=app /go/src/handler/handler.bin  handler
COPY --from=build --chown=app /usr/bin/fwatchdog  .

USER app

ENV fprocess="./handler"
ENV mode="http"
ENV upstream_url="http://127.0.0.1:8082"
ENV prefix_logs="false"
ENV ready_path="/_/ready"

CMD ["./fwatchdog"]
