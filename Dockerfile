from golang:1.20-bullseye as builder

ENV CGO_ENABLED=0

WORKDIR /opt

COPY . /opt/

RUN go mod tidy

RUN go build .

FROM scratch

COPY --from=builder /opt/s3-http-proxy /s3-http-proxy

ENTRYPOINT ["/s3-http-proxy"]