package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"netcat-go/netcat/util"
	"os"
	"strings"
	"time"
)

func udpToWriter(conn *net.UDPConn, out io.Writer, listen bool) <-chan net.UDPAddr {
	buf := make([]byte, 1024)
	syncChannel := make(chan net.UDPAddr)
	go func() {
		defer close(syncChannel)
		var remoteAddr *net.UDPAddr
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("read from udp failed: %s\n", err)
				}
				break
			}
			data := string(buf[:n])
			if listen && remoteAddr == nil {
				if !strings.HasPrefix(data, "olleh") {
					fmt.Printf("from %s: %s\n", addr, data)
				} else {
					remoteAddr = addr
					syncChannel <- *remoteAddr
				}
				continue
			}
			_, err = out.Write([]byte(data))
			if err != nil {
				break
			}
			if strings.HasPrefix(strings.ToLower(data), "exit") {
				break
			}
		}
	}()
	return syncChannel
}

func readerToUdp(in io.Reader, conn *net.UDPConn, clientAddr net.UDPAddr, listen bool) <-chan net.UDPAddr {
	syncChannel := make(chan net.UDPAddr)
	go func() {
		defer close(syncChannel)
		reader := bufio.NewReader(in)
		first := true
		for {
			if first && !listen {
				_, err := conn.Write([]byte("olleh"))
				if err != nil {
					log.Printf("write failed: %s\n", err)
					break
				}
				first = false
			}
			text, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("read error: %s\n", err)
				break
			}
			if listen {
				_, err = conn.WriteToUDP([]byte(text), &clientAddr)
			} else {
				_, err = conn.Write([]byte(text))
			}
			if err != nil {
				log.Printf("write to [%s] failed: %s", clientAddr.String(), err)
				break
			}
			if strings.HasPrefix(strings.ToLower(text), "exit") {
				break
			}
		}
	}()
	return syncChannel
}

func udpServerHandle(conn *net.UDPConn) {
	in_channel := udpToWriter(conn, os.Stdout, true)
	clientAddr := <-in_channel
	fmt.Printf("from %s\n", clientAddr.String())
	out_channel := readerToUdp(os.Stdin, conn, clientAddr, true)
	select {
	case <-in_channel:
		fmt.Printf("connect closed\n")
	case <-out_channel:
		fmt.Printf("local terminated\n")
	}
}

func udpClientHandle(conn *net.UDPConn, serverAddr *net.UDPAddr, r io.Reader) {
	out_channel := readerToUdp(r, conn, *serverAddr, false)
	in_channel := udpToWriter(conn, os.Stdout, false)
	select {
	case <-in_channel:
		fmt.Printf("connect closed\n")
	case <-out_channel:
		fmt.Printf("local terminated\n")
	}
}

func main() {
	ip := flag.String("h", "127.0.0.1", "server addr")
	port := flag.Int("p", 9300, "listen port")
	is_server := flag.Bool("l", false, "true: is server")
	count := flag.Int("c", 0, "send count")
	message := flag.String("m", "hello world\n", "message send to server")
	interval := flag.Int("i", 1, "client send interval")

	flag.Parse()
	if !flag.Parsed() {
		log.Fatalf("flag is not parsed\n")
	}
	if *is_server {
		conn, err := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: *port,
		})
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		udpServerHandle(conn)
	} else {
		addr, err := net.ResolveIPAddr("ip", *ip)
		if err != nil {
			log.Fatalf("parse addr[%s] failed: %s\n", *ip, err)
		}
		remote := net.UDPAddr{
			IP:   addr.IP,
			Port: *port,
		}
		conn, err := net.DialUDP("udp4", nil, &remote)
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		if *count == 0 {
			udpClientHandle(conn, &remote, os.Stdin)
		} else {
			sw := util.MakeStringWriter()
			go func() {
				defer sw.Close()
				for i := 0; i < *count; i++ {
					sw.WriteString(*message)
					fmt.Printf("%d: write %s\n", i, *message)
					time.Sleep(time.Duration(*interval) * time.Second)
				}
			}()
			udpClientHandle(conn, &remote, sw)
		}
	}
}
