package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(">")
		string, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}

		_, err = conn.Write([]byte(string))
		if err != nil {
			fmt.Println(err)
		}
	}
}
