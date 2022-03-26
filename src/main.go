package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
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
	Calls the apprpriate handle_connections based on tunnel mode
*/
func handleConnection(conn net.Conn, tunnel bool) {
	if tunnel {
		handleTunnelConnection(conn)
	} else if address, err := handleSocksConnection(conn); err == nil {
		serverResponse(conn, address)
	}
}

/*
	Detect the addresses which can  be used for dispatching in non-tunnelling mode.
	Alternate to ipconfig/ifconfig
*/
func detectInterfaces() {
	fmt.Println("--- Listing the available adresses for dispatching")
	ifaces, _ := net.Interfaces()

	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp == net.FlagUp) && (iface.Flags&net.FlagLoopback != net.FlagLoopback) {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						fmt.Printf("[+] %s, IPv4:%s\n", iface.Name, ipnet.IP.String())
					}
				}
			}
		}
	}

}

/*
	Gets the interface associated with the IP
*/
func getIfaceFromIp(ip string) string {
	ifaces, _ := net.Interfaces()

	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp == net.FlagUp) && (iface.Flags&net.FlagLoopback != net.FlagLoopback) {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						if ipnet.IP.String() == ip {
							return iface.Name + "\x00"
						}
					}
				}
			}
		}
	}
	return ""
}

/*
	Parses the command line arguements to obtain the list of load balancers
*/
func parseLoadBalancers(args []string, tunnel bool) {
	if len(args) == 0 {
		log.Fatal("[FATAL] Please specify one or more load balancers")
	}

	lbList = make([]LoadBalancer, flag.NArg())

	for idx, a := range args {
		splitted := strings.Split(a, "@")
		iface := ""
		var lbIp string
		var lbPort int
		var err error

		if tunnel {
			ipPort := strings.Split(splitted[0], ":")
			if len(ipPort) != 2 {
				log.Fatal("[FATAL] Invalid address specification ", splitted[0])
				return
			}

			lbIp = ipPort[0]
			lbPort, err = strconv.Atoi(ipPort[1])
			if err != nil || lbPort <= 0 || lbPort > 65535 {
				log.Fatal("[FATAL] Invalid port ", splitted[0])
				return
			}

		} else {
			lbIp = splitted[0]
			lbPort = 0
		}

		if net.ParseIP(lbIp).To4() == nil {
			log.Fatal("[FATAL] Invalid address ", lbIp)
		}

		contRatio := 1
		if len(splitted) > 1 {
			contRatio, err = strconv.Atoi(splitted[1])
			if err != nil || contRatio <= 0 {
				log.Fatal("[FATAL] Invalid contention ratio for ", lbIp)
			}
		}

		// Obtaining the interface name of the load balancer IP's doesn't make sense in tunnel mode
		if !tunnel {
			iface = getIfaceFromIp(lbIp)
			if iface == "" {
				log.Fatal("[FATAL] IP address not associated with an interface ", lbIp)
			}
		}

		log.Printf("[INFO] Load balancer %d: %s, contention ratio: %d\n", idx+1, lbIp, contRatio)
		lbList[idx] = LoadBalancer{address: fmt.Sprintf("%s:%d", lbIp, lbPort), iface: iface, contentionRatio: contRatio, currentConnections: 0}
	}
}

/*
	Main function
*/
func main() {
	var lhost = flag.String("lhost", "127.0.0.1", "The host to listen for SOCKS connection")
	var lport = flag.Int("lport", 8080, "The local port to listen for SOCKS connection")
	var detect = flag.Bool("list", false, "Shows the available addresses for dispatching (non-tunnelling mode only)")
	var tunnel = flag.Bool("tunnel", false, "Use tunnelling mode (acts as a transparent load balancing proxy)")

	flag.Parse()
	if *detect {
		detectInterfaces()
		return
	}

	// Disable timestamp in log messages
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// Check for valid IP
	if net.ParseIP(*lhost).To4() == nil {
		log.Fatal("[FATAL] Invalid host ", *lhost)
	}

	// Check for valid port
	if *lport < 1 || *lport > 65535 {
		log.Fatal("[FATAL] Invalid port ", *lport)
	}

	//Parse remaining string to get addresses of load balancers
	parseLoadBalancers(flag.Args(), *tunnel)

	localBindAddress := fmt.Sprintf("%s:%d", *lhost, *lport)

	// Start local server
	l, err := net.Listen("tcp4", localBindAddress)
	if err != nil {
		log.Fatalln("[FATAL] Could not start local server on ", localBindAddress)
	}
	log.Println("[INFO] Local server started on ", localBindAddress)
	//goland:noinspection GoUnhandledErrorResult
	defer l.Close()

	mutex = &sync.Mutex{}
	for {
		conn, _ := l.Accept()
		go handleConnection(conn, *tunnel)
	}
}
