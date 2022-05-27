package network

import (
	"githum.com/jiangzuomin/log"
	"net"
	"sync"
	"time"
)

type TCPServer struct{
	Addr string
	MaxConnNum int
	PendingWriteNum int
	NewAgent func(*TCPConn) Agent
	ln net.Listener
	conns ConnSet

}
