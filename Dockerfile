FROM golang:1.17-alpine as build

COPY ./src /tmp/src
WORKDIR /tmp/src
RUN go build -o /tmp/gobuilt


# TODO Use scratch image
FROM alpine:3.6

COPY --from=build /tmp/gobuilt /socks-proxy-balancer
RUN chmod +x /socks-proxy-balancer
ENTRYPOINT ["/socks-proxy-balancer"]
