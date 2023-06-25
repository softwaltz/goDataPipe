package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

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
			go pipeLine(laddr, raddr)
		}
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	<-s
	fmt.Println("Stop Piping!!!")
}

func pipeLine(listenAddr, remoteAddr string) {
	fmt.Println(listenAddr, "<->", remoteAddr)
	lsock, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println("Failed to listen:", err)
		os.Exit(1)
	}
	defer lsock.Close()

	copyIO := func(src, dest net.Conn) {
		defer src.Close()
		defer dest.Close()
		io.Copy(src, dest)
	}

	for {
		conn, err := lsock.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}

		proxy, err := net.Dial("tcp", remoteAddr)
		if err != nil {
			fmt.Println("Failed to connect to remote host:", err)
			conn.Close()
			continue
		}

		go copyIO(conn, proxy)
		go copyIO(proxy, conn)
	}
}
