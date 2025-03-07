ARG WATCHDOG_IMAGE_REPO=ghcr.io/openfaas/of-watchdog
ARG WATCHDOG_IMAGE_TAG=0.10.6
ARG GO_IMAGE_REPO=golang
ARG GO_IMAGE_TAG=1.23-alpine3.21
ARG FINAL_BASE_IMAGE=alpine:3.21

FROM --platform=${TARGETPLATFORM:-linux/amd64} ${WATCHDOG_IMAGE_REPO}:${WATCHDOG_IMAGE_TAG} AS watchdog
FROM --platform=${BUILDPLATFORM:-linux/amd64} ${GO_IMAGE_REPO}:${GO_IMAGE_TAG} AS build

RUN go install -v github.com/incu6us/goimports-reviser/v3@v3.8.2
RUN go install golang.org/x/tools/cmd/goimports@6a5b66bef78dc7a1cf8593b276f35102ec0cb11c

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
FROM --platform=${TARGETPLATFORM:-linux/amd64} ${FINAL_BASE_IMAGE} AS ship

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
ENV exec_timeout="0"
ENV read_timeout="300"
ENV write_timeout="300"

CMD ["./fwatchdog"]
