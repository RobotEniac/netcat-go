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

type Message struct {
	data string
	addr *net.UDPAddr
}

func main() {
	port := flag.Int("p", 9300, "listen port")
	flag.Parse()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		Port: *port,
		IP:   net.ParseIP("0.0.0.0"),
	})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Printf("listening at %s\n", conn.LocalAddr().String())
	udpChan := make(chan string, 1)
	stdinChan := make(chan string, 1)
	reader := bufio.NewReader(os.Stdin)
	go func() {
		defer close(udpChan)
		for {
			message := make([]byte, 1024)
			rlen, addr, err := conn.ReadFromUDP(message)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("client abort: %s\n", err)
				}
				break
			}
			if rlen == 0 {
				fmt.Println("client exit")
				break
			}
			data := strings.TrimSpace(string(message[:rlen]))
			udpChan <- fmt.Sprintf("from %s: %s", addr.String(), data)
			if strings.HasPrefix(data, "exit") {
				break
			}
		}
	} ()
	go func () {
		defer close(stdinChan)
		for {
			text, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("read error: %s\n", err)
				break
			}
			stdinChan <- strings.Trim(text, "\n")
			if strings.HasPrefix(strings.ToLower(text), "exit") {
				break
			}
		}
	} ()
	var clientAddr *net.UDPAddr
	for clientAddr == nil {
		tmp := make([]byte, 32)
		tmp_len, tmpAddr, err := conn.ReadFromUDP(tmp)
		if err != nil {
			log.Fatalf("get client addr failed: %s", err)
		}
		tt := string(tmp[:tmp_len])
		if strings.HasPrefix(tt, "olleh") {
			clientAddr = tmpAddr
		}
		fmt.Printf("conn from %s\n", clientAddr)
	}
	loop:
	for {
		select {
		case msg, ok := <- udpChan:
			if !ok {
				break loop
			}
			fmt.Println(msg)
			if strings.HasPrefix(msg, "exit") {
				break loop
			}
		case text, ok := <-stdinChan:
			if !ok {
				break loop
			}
			wlen, err := conn.WriteToUDP([]byte(text), clientAddr)
			if err != nil {
				fmt.Printf("write to udp failed: %s\n", err)
				break loop
			}
			if wlen == 0 {
				fmt.Printf("cannot write any more\n")
				break loop
			}
		}
	}
}
