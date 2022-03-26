package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
)

/*

 */
func clientGreeting(conn net.Conn) (byte, []byte, error) {
	buf := make([]byte, 2)

	if nRead, err := conn.Read(buf); err != nil || nRead != len(buf) {
		return 0, nil, errors.New("[WARN] client greeting failed")
	}

	socksVersion := buf[0]
	numAuthMethods := buf[1]

	authMethods := make([]byte, numAuthMethods)

	if nRead, err := conn.Read(authMethods); err != nil || nRead != int(numAuthMethods) {
		return 0, nil, errors.New("[WARN] client greeting failed")
	}

	return socksVersion, authMethods, nil
}

/*

 */
func serversChoice(conn net.Conn) error {

	if nWrite, err := conn.Write([]byte{5, 0}); err != nil || nWrite != 2 {
		return errors.New("[WARN] servers choice failed")
	}
	return nil
}

/*

 */
func clientConectionRequest(conn net.Conn) (string, error) {
	header := make([]byte, 4)
	port := make([]byte, 2)
	var address string

	//goland:noinspection GoUnhandledErrorResult
	if nRead, err := conn.Read(header); err != nil || nRead != len(header) {
		conn.Write([]byte{5, RequestStatusServerFailure, 0, 1, 0, 0, 0, 0, 0, 0})
		conn.Close()
		return "", errors.New("[WARN] client connection request failed")
	}

	socksVersion := header[0]
	cmdCode := header[1]
	//	reserved := header[2]
	addressType := header[3]

	//goland:noinspection GoUnhandledErrorResult
	if socksVersion != 5 {
		conn.Write([]byte{5, RequestStatusServerFailure, 0, 1, 0, 0, 0, 0, 0, 0})
		conn.Close()
		return "", errors.New("[WARN] unsupported SOCKS version")
	}

	//goland:noinspection GoUnhandledErrorResult
	if cmdCode != CommandConnect {
		conn.Write([]byte{5, RequestStatusCommandNotSupported, 0, 1, 0, 0, 0, 0, 0, 0})
		conn.Close()
		return "", errors.New("[WARN] unsupported command code")
	}

	//goland:noinspection GoUnhandledErrorResult
	switch addressType {
	case AddrTypeIPv4:
		ipv4Address := make([]byte, 4)

		if nRead, err := conn.Read(ipv4Address); err != nil || nRead != len(ipv4Address) {
			conn.Write([]byte{5, RequestStatusServerFailure, 0, 1, 0, 0, 0, 0, 0, 0})
			conn.Close()
			return "", errors.New("[WARN] client connection request failed")
		}

		if nRead, err := conn.Read(port); err != nil || nRead != len(port) {
			conn.Write([]byte{5, RequestStatusServerFailure, 0, 1, 0, 0, 0, 0, 0, 0})
			conn.Close()
			return "", errors.New("[WARN] client connection request failed")
		}
		address = fmt.Sprintf("%d.%d.%d.%d:%d", ipv4Address[0],
			ipv4Address[1],
			ipv4Address[2],
			ipv4Address[3],
			binary.BigEndian.Uint16(port))

	case AddrTypeDomain:
		domainNameLength := make([]byte, 1)

		if nRead, err := conn.Read(domainNameLength); err != nil || nRead != len(domainNameLength) {
			conn.Write([]byte{5, RequestStatusServerFailure, 0, 1, 0, 0, 0, 0, 0, 0})
			conn.Close()
			return "", errors.New("[WARN] client connection request failed")
		}

		domainName := make([]byte, domainNameLength[0])

		if nRead, err := conn.Read(domainName); err != nil || nRead != len(domainName) {
			conn.Write([]byte{5, RequestStatusServerFailure, 0, 1, 0, 0, 0, 0, 0, 0})
			conn.Close()
			return "", errors.New("[WARN] client connection request failed")
		}

		if nRead, err := conn.Read(port); err != nil || nRead != len(port) {
			conn.Write([]byte{5, RequestStatusServerFailure, 0, 1, 0, 0, 0, 0, 0, 0})
			conn.Close()
			return "", errors.New("[WARN] client connection request failed")
		}
		address = fmt.Sprintf("%s:%d", string(domainName), binary.BigEndian.Uint16(port))

	default:
		conn.Write([]byte{5, RequestStatusAddrtypeNotSupported, 0, 1, 0, 0, 0, 0, 0, 0})
		conn.Close()
		return "", errors.New("[WARN] unsupported address type")
	}
	return address, nil
}

/*

 */
func handleSocksConnection(conn net.Conn) (string, error) {

	if _, _, err := clientGreeting(conn); err != nil {
		log.Println(err)
		return "", err
	}

	if err := serversChoice(conn); err != nil {
		log.Println(err)
		return "", err
	}

	address, err := clientConectionRequest(conn)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return address, nil
}
