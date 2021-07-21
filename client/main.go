package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	ip := flag.String("h", "127.0.0.1", "server ip")
	port := flag.Int("p", 9300, "server port")
	flag.Parse()
	if !flag.Parsed() {
		log.Fatalf("flag parse failed")
	}
	remote := net.UDPAddr{
		IP:   net.ParseIP(*ip),
		Port: *port,
	}
	conn, err := net.DialUDP("udp", nil, &remote)
	if err != nil {
		fmt.Printf("Some error %v", err)
		return
	}
	defer conn.Close()
	_, err = conn.Write([]byte("olleh"))
	if err != nil  {
		log.Fatalf("write failed: %s\n", err)
	}
	udpChan := make(chan string, 1)
	stdinChan := make(chan string, 1)
	reader := bufio.NewReader(os.Stdin)
	go func() {
		defer close(stdinChan)
		for {
			text, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Printf("read error: %s\n", err)
				}
				break
			}
			stdinChan <- text
			if strings.HasPrefix(text, "exit") {
				break
			}
		}
	} ()
	go func() {
		defer close(udpChan)
		for {
			buf := make([]byte, 1024)
			rlen, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Printf("read from udp failed: %s\n", err)
				break
			}
			if rlen == 0 {
				fmt.Printf("read empty\n")
				break
			}
			data := strings.Trim(string(buf[:rlen]), "\n")
			udpChan <- data
			if strings.HasPrefix(data, "exit") {
				break
			}
		}
	} ()

	loop:
	for {
		select {
		case udpIn, ok := <- udpChan:
			if !ok {
				break loop
			}
			fmt.Println(udpIn)
		case text, ok := <- stdinChan:
			if !ok {
				break loop
			}
			wlen, err := conn.Write([]byte(text))
			if err != nil {
				fmt.Printf("write to udp failed: %s\n", err)
				break loop
			}
			if wlen == 0 {
				fmt.Printf("cannot write any more\n")
				break loop
			}
			if strings.HasPrefix(strings.ToLower(text), "exit") {
				break loop
			}
		}
	}
}
