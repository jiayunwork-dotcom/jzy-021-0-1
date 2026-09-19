# syntax=docker/dockerfile:1
FROM golang:1.22 AS builder
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /bin/cpmserver .

FROM alpine:3.20
RUN adduser -D -u 10001 cpm && mkdir -p /data && chown cpm /data
USER cpm
COPY --from=builder /bin/cpmserver /usr/local/bin/cpmserver
ENV CPM_ADDR=:8080 \
    CPM_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["cpmserver"]
