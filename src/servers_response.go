package main

import (
	"fmt"
	"log"
	"net"
	"syscall"
)

/*
	Implements servers response of SOCKS5 for linux systems
*/
func serverResponse(localConn net.Conn, remoteAddress string) {
	loadBalancer := getLoadBalancer()
	localTcpaddr, _ := net.ResolveTCPAddr("tcp4", loadBalancer.address)

	dialer := net.Dialer{
		LocalAddr: localTcpaddr,
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				// NOTE: Run with root or use setcap to allow interface binding
				// sudo setcap cap_net_raw=eip ./go-dispatch-proxy
				if err := syscall.BindToDevice(int(fd), loadBalancer.iface); err != nil {
					log.Println("[WARN] Couldn't bind to interface", loadBalancer.iface)
				}
			})
		},
	}

	remoteConn, err := dialer.Dial("tcp4", remoteAddress)
	//goland:noinspection GoUnhandledErrorResult
	if err != nil {
		log.Println("[WARN]", remoteAddress, "->", loadBalancer.address, fmt.Sprintf("{%s}", err))
		localConn.Write([]byte{5, RequestStatusNetworkUnreachable, 0, 1, 0, 0, 0, 0, 0, 0})
		localConn.Close()
		return
	}

	log.Println("[DEBUG]", remoteAddress, "->", loadBalancer.address)
	//goland:noinspection GoUnhandledErrorResult
	localConn.Write([]byte{5, RequestStatusSuccess, 0, 1, 0, 0, 0, 0, 0, 0})
	pipeConnections(localConn, remoteConn)
}
