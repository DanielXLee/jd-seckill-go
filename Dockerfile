# Build the manager binary
FROM golang:1.15.7-alpine3.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY config/ config/
COPY common/ common/
COPY seckill/ seckill/
COPY service/ service/
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Run
FROM alpine:3.13

RUN apk --no-cache add libqrencode zbar
WORKDIR /
COPY --from=builder /workspace/manager .
COPY conf.ini .

ENTRYPOINT ["/manager"]
