package udp

import (
	"fmt"
	"net"
)

func Server() {
	listen, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9090,
	})

	if err != nil {
		fmt.Printf("listen failed, err:%v\n", err)
		return
	}
	// 无限循坏处理udp的请求
	for {
		var data [1024]byte
		n, addr, err := listen.ReadFromUDP(data[:])
		if err != nil {
			fmt.Printf("read failed from addr: %v, err: %v\n", addr, err)
			break
		}

		fmt.Printf("addr: %v data: %v  count: %v\n", addr, string(data[:n]), n)

		_, err = listen.WriteToUDP([]byte("received success!"), addr)
		if err != nil {
			fmt.Printf("write failed, err: %v\n", err)
			continue
		}
	}
}

func Client() {
	//connect server
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 9090,
	})

	if err != nil {
		fmt.Printf("connect failed, err: %v\n", err)
		return
	}

	//send data
	_, err = conn.Write([]byte("hello server!"))
	if err != nil {
		fmt.Printf("send data failed, err : %v\n", err)
		return
	}

	//receive data from server
	result := make([]byte, 4096)
	n, remoteAddr, err := conn.ReadFromUDP(result)
	if err != nil {
		fmt.Printf("receive data failed, err: %v\n", err)
		return
	}
	fmt.Printf("receive from addr: %v  data: %v\n", remoteAddr, string(result[:n]))
}
