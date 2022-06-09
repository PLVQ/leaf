package cluster

import (
	 "github.com/jiangzuomin/leaf/config"
	 "github.com/jiangzuomin/leaf/network"
	 "math"
	 "time"
)

var (
	server  *network.TCPServer
	clients []*network.TCPClient
)

func Init() {
	if config.ListenAddr != "" {
		server = new(network.TCPServer)
		server.Addr = config.ListenAddr
		server.MaxConnNum = int(math.MaxInt32)
		server.PendingWriteNum = config.PendingWriteNum
		server.LenMsgLen = 4
		server.MaxMsgLen = math.MaxUint32
		server.NewAgent = newAgent

		server.Start()
	}

	for _, addr := range config.ConnAddrs {
		client := new(network.TCPClient)
		client.Addr = addr
		client.ConnNum = 1
		client.ConnectInterval = 3 * time.Second
		client.PendingWriteNum = config.PendingWriteNum
		client.LenMsgLen = 4
		client.MaxMsgLen = math.MaxUint32
		client.NewAgent = newAgent

		client.Start()
		clients = append(clients, client)
	}
}

func Destroy() {
	if server != nil {
		server.Close()
	}

	for _, client := range clients {
		client.Close()
	}
}

type Agent struct {
	conn *network.TCPConn
}

func newAgent(conn *network.TCPConn) network.Agent {
	a := new(Agent)
	a.conn = conn
	return a
}

func (a *Agent) Run() {}

func (a *Agent) OnClose() {}