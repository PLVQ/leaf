package network

import (
	"net"
	"sync"
	"time"

	"github.com/jiangzuomin/leaf/log"
	"github.com/sirupsen/logrus"
)

type TCPClient struct {
	sync.Mutex
	Addr            string
	ConnNum         int
	ConnectInterval time.Duration
	PendingWriteNum int
	AutoReconnect   bool
	NewAgent        func(*TCPConn) Agent
	conns           ConnSet
	wg              sync.WaitGroup
	closeFlag       bool

	// msg parser
	LenMsgLen    int
	MinMsgLen    uint32
	MaxMsgLen    uint32
	LittleEndian bool
	msgParser    *MsgParser
}

func (client *TCPClient) init() {
	client.Lock()
	defer client.Unlock()

	if client.ConnNum <= 0 {
		client.ConnNum = 1
		log.Log.WithField("ConnNum", client.ConnNum).Info("Invalid ConnNum reset")
	}

	if client.ConnectInterval <= 0 {
		client.ConnectInterval = 3 * time.Second
		log.Log.WithField("ConnectInterval", client.ConnectInterval).Info("Invalid ConnectInterval reset")
	}

	if client.PendingWriteNum <= 0 {
		client.PendingWriteNum = 100
		log.Log.WithField("PendingWriteNum", client.PendingWriteNum).Info("Invalid PendingWriteNum reset")
	}

	if client.NewAgent == nil {
		log.Log.Fatal("NewAgent must not be nil")
	}
	if client.conns != nil {
		log.Log.Fatal("client is running")
	}

	client.conns = make(ConnSet)
	client.closeFlag = false

	msgParser := NewMsgParser()
	msgParser.SetMsgLen(client.LenMsgLen, client.MinMsgLen, client.MaxMsgLen)
	msgParser.SetByteOrder(client.LittleEndian)

	client.msgParser = msgParser
}

func (client *TCPClient) dial() net.Conn {
	for {
		conn, err := net.Dial("tcp", client.Addr)
		if err == nil || client.closeFlag {
			return conn
		}

		log.Log.WithFields(logrus.Fields{"Addr": client.Addr, "Error": err}).Error("Connect error")
		time.Sleep(client.ConnectInterval)
		continue
	}
}

func (client *TCPClient) connect() {
	defer client.wg.Done()
reconnect:
	conn := client.dial()
	if conn == nil {
		return
	}

	client.Lock()

	if client.closeFlag {
		client.Unlock()
		conn.Close()
		return
	}

	client.conns[conn] = struct{}{}
	client.Unlock()

	tcpConn := newTCPConn(conn, client.PendingWriteNum, client.msgParser)
	agent := client.NewAgent(tcpConn)
	agent.Run()

	tcpConn.Close()

	client.Lock()
	delete(client.conns, conn)
	client.Unlock()
	agent.OnClose()

	if client.AutoReconnect {
		time.Sleep(client.ConnectInterval)
		goto reconnect
	}
}

func (client *TCPClient) Close() {
	client.Lock()
	client.closeFlag = true
	for conn := range client.conns {
		conn.Close()
	}
	client.conns = nil
	client.Unlock()

	client.wg.Wait()
}

func (client *TCPClient) Start() {
	client.init()

	for i := 0; i < client.ConnNum; i++ {
		client.wg.Add(1)
		go client.connect()
	}
}
