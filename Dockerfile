FROM golang:1.14 as build
COPY udpproxy.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o udpproxy .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
ENV ENTRYPOINT ""
ENV LISTEN_PORT ""
COPY --from=build /go/udpproxy .
CMD ["./udpproxy"]
