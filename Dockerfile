FROM golang:1.16.3-alpine3.13 as builder

WORKDIR /go/src/github.com/rkrmr33/template-hub

RUN apk -U add --no-cache git make ca-certificates openssl && update-ca-certificates

# See https://stackoverflow.com/a/55757473/12429735RUN 
RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/home/template-hub" \    
    --shell "/sbin/nologin" \    
    --no-create-home \    
    --uid "10001" \    
    "template-hub"


# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download -x

# Copy and build binary
COPY . .

RUN make

FROM gcr.io/distroless/base AS template-hub

WORKDIR /template-hub

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /go/src/github.com/rkrmr33/template-hub/dist/server-* server

USER template-hub:template-hub

ENTRYPOINT ["/template-hub/server"]
