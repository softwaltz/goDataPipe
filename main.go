package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

const (
	MAXCLIENTS  = 20
	IDLETIMEOUT = 300
	BUFFER_SIZE = 4096
)

type client struct {
	inuse    bool
	csock    net.Conn
	osock    net.Conn
	activity time.Time
}

func main() {
	if len(os.Args) != 5 {
		fmt.Printf("Usage: %s localhost localport remotehost remoteport\n", os.Args[0])
		os.Exit(1)
	}

	listenAddr := os.Args[1] + ":" + os.Args[2]
	remoteAddr := os.Args[3] + ":" + os.Args[4]

	laddr, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		fmt.Println("Failed to resolve listen address:", err)
		os.Exit(1)
	}

	raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		fmt.Println("Failed to resolve remote address:", err)
		os.Exit(1)
	}

	lsock, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		fmt.Println("Failed to listen:", err)
		os.Exit(1)
	}
	defer lsock.Close()

	clients := make([]client, MAXCLIENTS)

	for {
		conn, err := lsock.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}

		i := findFreeClient(clients)
		if i < 0 {
			fmt.Println("Too many clients")
			conn.Close()
			continue
		}

		osock, err := net.DialTCP("tcp", nil, raddr)
		if err != nil {
			fmt.Println("Failed to connect to remote host:", err)
			conn.Close()
			continue
		}

		clients[i] = client{
			inuse:    true,
			csock:    conn,
			osock:    osock,
			activity: time.Now(),
		}

		go handleClient(&clients[i])
	}
}

func handleClient(cli *client) {
	buf := make([]byte, BUFFER_SIZE)

	for {
		// Set read and write deadlines to detect idle clients
		cli.csock.SetReadDeadline(time.Now().Add(IDLETIMEOUT * time.Second))
		cli.osock.SetWriteDeadline(time.Now().Add(IDLETIMEOUT * time.Second))

		n, err := cli.csock.Read(buf)
		if err != nil {
			cli.csock.Close()
			cli.osock.Close()
			cli.inuse = false
			return
		}

		_, err = cli.osock.Write(buf[:n])
		if err != nil {
			cli.csock.Close()
			cli.osock.Close()
			cli.inuse = false
			return
		}

		cli.activity = time.Now()
	}
}

func findFreeClient(clients []client) int {
	for i, cli := range clients {
		if !cli.inuse {
			return i
		}
	}
	return -1
}
