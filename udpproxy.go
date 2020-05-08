package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

type Connection struct {
	ClientAddr  *net.UDPAddr
	RelayedConn *net.UDPConn
}

var (
	listenPort int
	endpoint   string
	proxyConn  *net.UDPConn
	clientMap  = make(map[string]*Connection)
)

func NewConnection(srvAddr, cliAddr *net.UDPAddr) *Connection {
	srvUDP, err := net.DialUDP("udp", nil, srvAddr)
	if err != nil {
		return nil
	}

	conn := &Connection{
		ClientAddr:  cliAddr,
		RelayedConn: srvUDP,
	}

	return conn
}

func ProxyLoop(conn *Connection) {
	var buffer [1500]byte
	for {
		n, err := conn.RelayedConn.Read(buffer[0:])
		if err != nil {
			continue
		}

		_, err = proxyConn.WriteToUDP(buffer[0:n], conn.ClientAddr)
		if err != nil {
			continue
		}
		fmt.Printf("OUT: %d bytes from server to %s.\n", n, conn.ClientAddr.String())
	}
}

func main() {
	flag.IntVar(&listenPort, "listen-port", lookupEnvOrInt("LISTEN_PORT", listenPort), "port to listen")
	flag.StringVar(&endpoint, "endpoint", lookupEnvOrString("ENDPOINT", endpoint), "endpoint")

	flag.Parse()
	log.Println("app.status=starting")
	defer log.Println("app.status=shutdown")

	log.Printf("app.config %v\n", getConfig(flag.CommandLine))

	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		log.Fatal(err)
	}
	serverConnection, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	proxyConn = serverConnection
	endpointAddr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		log.Fatal(err)
	}
	mutex := new(sync.Mutex)
	var buffer [1500]byte
	for {
		n, cliAddr, err := proxyConn.ReadFromUDP(buffer[0:])
		if err != nil {
			continue
		}
		sCliAddr := cliAddr.String()
		mutex.Lock()
		conn, found := clientMap[sCliAddr]
		if !found {
			conn = NewConnection(endpointAddr, cliAddr)
			if conn == nil {
				mutex.Unlock()
				continue
			}
			clientMap[sCliAddr] = conn
			go ProxyLoop(conn)
		}
		mutex.Unlock()
		// Relay to server
		_, err = conn.RelayedConn.Write(buffer[0:n])
		if err != nil {
			continue
		}
	}
}

func lookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func lookupEnvOrInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("lookupEnvOrInt[%s]: %v", key, err)
		}
		return v
	}
	return defaultVal
}

func getConfig(fs *flag.FlagSet) []string {
	cfg := make([]string, 0, 10)
	fs.VisitAll(func(f *flag.Flag) {
		cfg = append(cfg, fmt.Sprintf("%s:%q", f.Name, f.Value.String()))
	})

	return cfg
}
