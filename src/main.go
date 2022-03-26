package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

type LoadBalancer struct {
	address            string
	iface              string
	contentionRatio    int
	currentConnections int
}

// The load balancer used in the previous connection
var lbIndex int

// List of all load balancers
var lbList []LoadBalancer

// Mutex to serialize access to function getLoadBalancer
var mutex *sync.Mutex

/*
	Get a load balancer according to contention ratio
*/
func getLoadBalancer() *LoadBalancer {
	mutex.Lock()
	lb := &lbList[lbIndex]
	lb.currentConnections += 1

	if lb.currentConnections == lb.contentionRatio {
		lb.currentConnections = 0
		lbIndex += 1

		if lbIndex == len(lbList) {
			lbIndex = 0
		}
	}
	mutex.Unlock()
	return lb
}

/*
	Joins the local and remote connections together
*/
func pipeConnections(localConn, remoteConn net.Conn) {
	//goland:noinspection GoUnhandledErrorResult
	go func() {
		defer remoteConn.Close()
		defer localConn.Close()
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			return
		}
	}()

	//goland:noinspection GoUnhandledErrorResult
	go func() {
		defer remoteConn.Close()
		defer localConn.Close()
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			return
		}
	}()
}

/*
	Handle connections in tunnel mode
*/
func handleTunnelConnection(conn net.Conn) {
	loadBalancer := getLoadBalancer()

	remoteAddr, _ := net.ResolveTCPAddr("tcp4", loadBalancer.address)
	remoteConn, err := net.DialTCP("tcp4", nil, remoteAddr)

	if err != nil {
		log.Println("[WARN]", loadBalancer.address, fmt.Sprintf("{%s}", err))
		_ = conn.Close()
		return
	}

	log.Println("[DEBUG] Tunnelled to", loadBalancer.address)
	pipeConnections(conn, remoteConn)
}

/*
	Parses the command line arguements to obtain the list of load balancers
*/
func parseLoadBalancers(proxies []HostPort) {
	lbList = make([]LoadBalancer, len(proxies))

	for i, proxy := range proxies {
		contRatio := 1
		log.Printf("[INFO] Load balancer %d: %s:%d, contention ratio: %d\n", i+1, proxy.Host, proxy.Port, contRatio)
		lbList[i] = LoadBalancer{
			address:            proxy.ToAddr(),
			iface:              "",
			contentionRatio:    contRatio,
			currentConnections: 0,
		}
	}
}

/*
	Main function
*/
func mainLogic(settings *Settings) {
	// Disable timestamp in log messages
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// Parse load balancers from settings into LoadBalancer objects
	parseLoadBalancers(settings.ProxiesConnect)

	// Start local server
	localBindAddress := settings.ToAddr()
	l, err := net.Listen("tcp4", localBindAddress)
	if err != nil {
		log.Fatalln("[FATAL] Could not start local server on", localBindAddress)
	}
	log.Println("[INFO] Local server started on", localBindAddress)
	//goland:noinspection GoUnhandledErrorResult
	defer l.Close()

	mutex = &sync.Mutex{}
	for {
		conn, _ := l.Accept()
		go handleTunnelConnection(conn)
	}
}

func main() {
	settings, errs := LoadSettings()
	if len(errs) > 0 {
		fmt.Println("Errors in settings:")
		for _, err := range errs {
			fmt.Printf("\t- %s\n", err.Error())
		}
		os.Exit(1)
	}

	mainLogic(settings)
}
