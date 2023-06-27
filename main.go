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

const BUFFER_SIZE = 1024

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s [lhost]:lport:rhost:rport [lhost]:lport:rhost:rport ...\n", os.Args[0])
		os.Exit(1)
	}

	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		tmp := strings.Split(arg, ":")
		if len(tmp) == 4 {
			if tmp[0] == "" {
				tmp[0] = "0.0.0.0"
			}
			laddr := tmp[0] + ":" + tmp[1]
			raddr := tmp[2] + ":" + tmp[3]
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
	fmt.Println("forwarding:", listenAddr, "<->", remoteAddr)
	lsock, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Println("Failed to listen:", err)
		os.Exit(1)
	}
	defer lsock.Close()

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

		go redirectIO(conn, proxy)
		go redirectIO(proxy, conn)
	}
}

func redirectIO(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

	total := 0
	defer func() {
		fmt.Println(src.RemoteAddr(), "--[", total, "byets]-->", dst.RemoteAddr())
	}()

	buf := make([]byte, BUFFER_SIZE)
	for {
		// Set read and write deadlines to detect idle clients
		n, err := src.Read(buf)
		if err != nil {
			return
		}
		_, err = dst.Write(buf[:n])
		if err != nil {
			return
		}
		total += n
	}
}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	n, _ := io.Copy(src, dest)
	fmt.Println("Writing", n, "bytes from", src.RemoteAddr(), "to", dest.RemoteAddr())
}
