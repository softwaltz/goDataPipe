package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	MAXCLIENTS  = 300
	IDLETIMEOUT = 300
)

type client struct {
	inuse    bool
	inSock   net.Conn
	outSock  net.Conn
	activity time.Time
}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s lhost:lport:rhost:rport lhost:lport:rhost:rport ...\n", os.Args[0])
		os.Exit(1)
	}

	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		tmp := strings.Split(arg, ":")
		if len(tmp) == 4 {
			laddr := tmp[0] + ":" + tmp[1]
			raddr := tmp[2] + ":" + tmp[3]
			fmt.Println(i, ":", laddr, raddr)
			go pipeLine2(laddr, raddr)
		}
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	<-s
	fmt.Println("Stop Piping!!!")
}

func pipeLine2(listenAddr, remoteAddr string) {
	fmt.Println(listenAddr, "<->", remoteAddr)

	lsock, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println("Failed to listen:", err)
		os.Exit(1)
	}
	defer lsock.Close()

	clients := make([]client, MAXCLIENTS)

	for {
		inSock, err := lsock.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}

		i := findFreeClient(clients)
		if i < 0 {
			fmt.Println("Too many clients")
			inSock.Close()
			continue
		}

		outSock, err := net.Dial("tcp", remoteAddr)
		if err != nil {
			fmt.Println("Failed to connect to remote host:", err)
			inSock.Close()
			continue
		}

		clients[i] = client{
			inuse:    true,
			inSock:   inSock,
			outSock:  outSock,
			activity: time.Now(),
		}

		handleClient(&clients[i])
	}
}

func handleClient(cli *client) {
	copyIO := func(src, dst net.Conn) {
		src.SetReadDeadline(time.Now().Add(IDLETIMEOUT * time.Second))
		dst.SetWriteDeadline(time.Now().Add(IDLETIMEOUT * time.Second))
		defer src.Close()
		defer dst.Close()
		io.Copy(src, dst)
	}

	cli.activity = time.Now()
	go copyIO(cli.inSock, cli.outSock)
	go copyIO(cli.outSock, cli.inSock)
}

func findFreeClient(clients []client) int {
	for i, cli := range clients {
		if !cli.inuse {
			return i
		}
	}
	return -1
}
