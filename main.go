package main

import (
	//	"bufio"
	//"container/ring"
	"bytes"
	"context"
	"fmt"
	"strconv"
	//	"io"
	"net"
	//	"os"
	//	"strings"
	"time"
)

func main() {
	fmt.Println("Starting the server ...")
	// 创建 listener
	listener, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		fmt.Println("Error listening", err.Error())
		return //终止程序
	}
	// 监听并接受来自客户端的连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting", err.Error())
			return // 终止程序
		}
		go doServerStuff(conn)
	}
}

func doServerStuff(conn net.Conn) {
	//header := ""
	//endCnt := 0
	var timeout time.Duration
	timeout = 10
	ctx1, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*timeout*1000))
	defer cancel()
	go func(ctx context.Context) {
		go dial(conn)
		cancel()
		//conn.Close()
	}(ctx1)

	select {
	case <-ctx1.Done():
		return
	case <-time.After(time.Duration(time.Millisecond*timeout*1000 + 100)):
		conn.Close()
		cancel()
		return
	}

}

func read(conn net.Conn) []byte {
	rs := []byte{}
	flag := false
	for {
		buf := make([]byte, 512)
		blen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading", err.Error())
			return []byte{}
		}
		ctx := buf[:blen]
		rs = append(rs, ctx...)
		if bytes.Equal(rs[len(rs)-4:], []byte{13, 10, 13, 10}) {
			flag = true
			break
		}
	}
	if flag {
		return rs
	}
	return []byte{}
}

func dial(conn net.Conn) {
	header := read(conn)
	if len(header) > 0 {
		b2m := func(h []byte) map[string]string {
			rs := map[string]string{}
			bs := bytes.Split(h, []byte{'\r', '\n'})
			for _, v := range bs {
				vv := bytes.SplitN(v, []byte{':', ' '}, 2)
				if len(vv) <= 1 {
					continue
				}
				rs[string(vv[0])] = string(vv[1])
			}
			return rs
		}
		header1 := b2m(header)
		if header1["Host"] == "gukube:8888" {
			ctxLenStr := header1["Content-Length"]
			ctxLen := -1
			if ctxLenStr != "" {
				ctxLen, _ = strconv.Atoi(ctxLenStr)
			}
			conn1, err := net.DialTimeout("tcp", "oa.xxynet.com:80", time.Second*10)
			if err != nil {
				fmt.Println("Error dialing", err.Error())
				return // 终止程序
			}
			_, err = conn1.Write(header)
			var endFunc func(a int, b []byte) bool
			if ctxLen >= 0 {
				endFunc = func(a int, b []byte) bool {
					return a >= ctxLen
				}
			} else {
				endFunc = func(a int, b []byte) bool {
					return bytes.Equal(b, []byte{13, 10, 0, 13, 10, 13, 10})
				}
			}
			go forword(conn, conn1, func(a int, b []byte) bool {
				return false
			})
			go forword(conn1, conn, endFunc)
		}
	}
}

func forword(c1, c2 net.Conn, endFunc func(int, []byte) bool) {
	buf := make([]byte, 512)
	defer c1.Close()
	defer c2.Close()
	lenEndpoint := 7
	endpoint := make([]byte, 0, 7)
	allLen := 0
	for {
		blen, err := c1.Read(buf)
		allLen += blen
		if err != nil {
			return
		}
		ctx := buf[:blen]
		c2.Write(ctx)
		if blen >= lenEndpoint {
			endpoint = ctx[blen-7:]
		} else {
			endpoint = append(endpoint[lenEndpoint-blen:], ctx...)
		}
		if endFunc(allLen, endpoint) {
			return
		}
		//endpoint = append(endpoint, ctx...)
	}
}
