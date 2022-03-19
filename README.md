# socks-proxy-balancer

A SOCKS5 load balancing proxy that combines multiple SOCKS5 proxies into one.

## Background

This project is forked from [go-dispatch-proxy](https://github.com/extremecoders-re/go-dispatch-proxy), with the following modifications/objectives:

- Focus on the feature of load-balancing multiple SOCKS5 proxies into one (which the original repository refers to as "SSH tunnel load balancing")
- Docker support
- Parametrization via env variables or config file instead of CLI (Docker friendly)
- Avoid logging each connection
