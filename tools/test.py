import os
import time
import subprocess
import threading
import requests


class Settings:
    def __init__(self):
        self.upstream_containers_count = int(os.getenv("UPSTREAM_CONTAINERS_COUNT", 4))
        if self.upstream_containers_count < 2:
            raise Exception("UPSTREAM_CONTAINERS_COUNT must be at least 2")

        self.network_name = os.getenv("NETWORK_NAME", "socks-proxy-balancer")
        self.proxy_balancer_container_name = os.getenv("PROXY_BALANCER_CONTAINER_NAME", "tor-proxy-balancer")
        self.proxy_balancer_container_image = os.getenv("PROXY_BALANCER_CONTAINER_IMAGE", "local/socks-proxy-balancer:latest")

        self.proxy_tor_container_name_prefix = os.getenv("PROXY_TOR_CONTAINER_NAME_PREFIX", "tor-proxy-")
        self.proxy_tor_container_image = os.getenv("PROXY_TOR_CONTAINER_IMAGE", "rdsubhas/tor-privoxy-alpine")
        self.proxy_port_base = int(os.getenv("PROXY_PORT_BASE", 9050))
        self.proxy_tor_container_socks_port = 9050
        self.proxy_tor_containers_names = list()
        self.proxy_tor_containers_host_ports = list()

    @property
    def docker_network_args(self):
        return ["--network", self.network_name]


def call(*args) -> str:
    print("Exec:", *args, "...")
    return subprocess.check_output(args).decode()


def setup_network(settings: Settings):
    call("docker", "network", "create", settings.network_name)


def setup_torproxy_upstream_containers(settings: Settings):
    for i0 in range(settings.upstream_containers_count):
        i1 = i0 + 1
        container_name = f"{settings.proxy_tor_container_name_prefix}{i1}"
        container_host_port = settings.proxy_port_base + i1
        settings.proxy_tor_containers_names.append(container_name)
        settings.proxy_tor_containers_host_ports.append(container_host_port)

        call(
            "docker", "run", "-d",
            "--name", container_name,
            "-p", f"{container_host_port}:{settings.proxy_tor_container_socks_port}",
            *settings.docker_network_args,
            settings.proxy_tor_container_image
        )


def setup_proxy_balancer(settings: Settings):
    container_port = settings.proxy_port_base
    env_upstream_proxy_servers = list()
    for i1, upstream_container_name in enumerate(settings.proxy_tor_containers_names, start=1):
        env_key = f"SOCKS_CONNECT_{i1}"
        env_value = f"{upstream_container_name}:{settings.proxy_tor_container_socks_port}"
        args = ["-e", f"{env_key}={env_value}"]
        env_upstream_proxy_servers.extend(args)

    call(
        "docker", "run", "-d",
        "--name", settings.proxy_balancer_container_name,
        "-p", f"{container_port}:{container_port}",
        *settings.docker_network_args,
        *env_upstream_proxy_servers,
        settings.proxy_balancer_container_image
    )


def get_public_ip(proxy: str) -> str:
    if proxy:
        proxy = f"socks5://{proxy}"
        proxy = {"http": proxy, "https": proxy}
    r = requests.get("https://api.ipify.org", proxies=proxy, timeout=10)
    r.raise_for_status()
    return r.text.strip()


def tst_get_ips_from_each_upstream_proxy(settings: Settings) -> set:
    """Request the public IP from each upstream tor-proxy server.
    Should end up with one unique IP per server."""
    ips = set()

    threads = list()
    for _ in range(settings.upstream_containers_count):
        proxy = f"localhost:{settings.proxy_tor_container_socks_port}"
        th = threading.Thread(
            target=lambda: ips.add(get_public_ip(proxy)),
            daemon=True
        )
        threads.append(th)
        th.start()

    [th.join() for th in threads]
    print("tst_get_ips_from_each_upstream_proxy:", ips)
    assert len(ips) == settings.upstream_containers_count
    return ips


def tst_get_ips_from_balancer_proxy(settings: Settings, upstream_proxies_ips: set):
    """Request the public IP from the balancer proxy multiple times.
    Should end up with the same amount of IPs as those returned previously,
    when querying the IP for each upstream proxy server."""
    threads_count = 20
    proxy = f"localhost:{settings.proxy_port_base}"

    ips = set()
    threads = list()
    for _ in range(threads_count):
        th = threading.Thread(
            target=lambda: ips.add(get_public_ip(proxy)),
            daemon=True
        )
        threads.append(th)
        th.start()

    [th.join() for th in threads]
    print("tst_get_ips_from_balancer_proxy:", ips)
    assert ips == upstream_proxies_ips


def teardown(settings: Settings):
    r = call(
        "docker", "container", "ls", "-q",
        "--filter", f"network={settings.network_name}"
    )
    # `r` is IDs of matching containers split by \n
    containers = r.splitlines()
    call("docker", "stop", *containers)
    call("docker", "rm", *containers)
    call("docker", "network", "rm", settings.network_name)


def main():
    settings = Settings()

    try:
        setup_network(settings)
        setup_torproxy_upstream_containers(settings)
        setup_proxy_balancer(settings)

        time.sleep(5)
        upstream_proxies_ips = tst_get_ips_from_each_upstream_proxy(settings)
        tst_get_ips_from_balancer_proxy(settings, upstream_proxies_ips)

    finally:
        teardown(settings)


if __name__ == '__main__':
    main()
