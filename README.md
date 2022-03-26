# socks-proxy-balancer

A SOCKS5 load balancing proxy that combines multiple SOCKS5 proxies into one.

## Background

This project is forked from [extremecoders-re/go-dispatch-proxy](https://github.com/extremecoders-re/go-dispatch-proxy), with the following modifications/objectives:

- Focus on the feature of load-balancing multiple SOCKS5 proxies into one (which the original repository refers to as "SSH tunnel load balancing")
- Docker support
- Linux-only support
- Parametrization via env variables or config file instead of CLI (Docker friendly)
- Avoid logging each connection

## Usage

The connection settings are configured using the following environment variables:

- `SOCKS_SERVER_PORT`: port used by the load-balancing SOCKS proxy server  (default: `9050`)
- `SOCKS_SERVER_HOST`: host/interface where the load-balancing SOCKS proxy server is bound; typical values are `0.0.0.0` to listen all interfaces, or `127.0.0.1` to listen on localhost only (default: `0.0.0.0`)

Definition of SOCKS proxies to forward requests to: you can define any number of environment variables, whose name start with `SOCKS_CONNECT`.
Their values are the target proxies in format `host:port` or `ip:port`. Multiple proxies can be given on a single variable, delimited by commas `,` (spaces are ignored).

For example, defining the following environment variables:

```dotenv
SOCKS_CONNECT="127.0.0.1:9051, 127.0.0.1:9052"
SOCKS_CONNECT_FOO="10.0.0.1:9050"
SOCKS_CONNECT2="10.0.0.2:9050,10.0.0.3:9051"
```

...will use ALL the defined servers (`127.0.0.1:9051, 127.0.0.1:9052, 10.0.0.1:9050, 10.0.0.2:9050, 10.0.0.3:9051`).

### Example: load balancing multiple Tor proxies using Docker

```bash
# Build the container
# (if make not available in your system, run the command referenced under "build:" in the Makefile)
make build

# Create a Docker network for this example
NETWORK="socks-proxy-balancer"
docker network create "$NETWORK"

# Start 4 tor-proxy containers;
# each container runs a SOCKS proxy server at port 9050, which forwards traffic through Tor
# (each proxy server should use a different public IP)
docker run -d --name=tor-proxy-1 -p 9051:9050 --network="$NETWORK" rdsubhas/tor-privoxy-alpine
docker run -d --name=tor-proxy-2 -p 9052:9050 --network="$NETWORK" rdsubhas/tor-privoxy-alpine
docker run -d --name=tor-proxy-3 -p 9053:9050 --network="$NETWORK" rdsubhas/tor-privoxy-alpine
docker run -d --name=tor-proxy-4 -p 9054:9050 --network="$NETWORK" rdsubhas/tor-privoxy-alpine

# Start the proxy balancer
docker run -d --name=tor-proxy-balancer -p 9050:9050 --network="$NETWORK" -e SOCKS_CONNECT="tor-proxy-1:9050, tor-proxy-2:9050, tor-proxy-3:9050, tor-proxy-4:9050" local/socks-proxy-balancer:latest

# Get the public IP from each tor-proxy container
curl -x socks5h://localhost:9051 https://api.ipify.org
curl -x socks5h://localhost:9052 https://api.ipify.org
curl -x socks5h://localhost:9053 https://api.ipify.org
curl -x socks5h://localhost:9054 https://api.ipify.org

# Get the public IP using the proxy balancer
# Call this command multiple times, each time should return a different IP
curl -x socks5h://localhost:9050 https://api.ipify.org

# Delete all
CONTAINERS=$(docker container ls -q --filter network=$NETWORK)
docker stop $CONTAINERS
docker rm $CONTAINERS
docker network rm "$NETWORK"
```
