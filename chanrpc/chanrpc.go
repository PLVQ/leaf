package chanrpc

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/jiangzuomin/leaf/config"
	"github.com/jiangzuomin/leaf/log"
	"github.com/sirupsen/logrus"
)

type Server struct {
	// id -> function
	//
	// function:
	// func(args []interface{})
	// func(args []interface{}) interface{}
	// func(args []interface{}) []interface{}
	functions map[interface{}]interface{}		// 注册函数映射
	ChanCall  chan *CallInfo					// 异步掉用一次性最多能传递多少函数
}

// 函数调用信息
type CallInfo struct {
	f       interface{}			// 函数
	args    []interface{}		// 函数参数
	chanRet chan *RetInfo		// 函数返回
	cb      interface{}			// 回调函数
}

// 函数返回信息
type RetInfo struct {
	ret interface{}				// 返回值
	err error					// 错误
	cb  interface{}				// 回调函数
}

type Client struct {
	s               *Server
	chanSyncRet     chan *RetInfo
	ChanAsynRet     chan *RetInfo
	PendingAsynCall int
}

func NewServer(chanCallLen int) *Server {
	server := new(Server)
	server.functions = make(map[interface{}]interface{})
	server.ChanCall = make(chan *CallInfo, chanCallLen)
	return server
}

func assert(i interface{}) []interface{} {
	if i == nil {
		return nil
	} else {
		return i.([]interface{})
	}
}

func (s *Server) Register(id interface{}, fc interface{}) {
	switch fc.(type) {
	case func([]interface{}):
	case func([]interface{}) interface{}:
	case func([]interface{}) []interface{}:
	default:
		log.Log.WithFields(logrus.Fields{"func id": id}).Fatal("definition of function is invalid")
	}

	if _, ok := s.functions[id]; ok {
		log.Log.WithField("func id", id).Fatal("function already registered")
	}

	s.functions[id] = fc
}

func (s *Server) ret(ci *CallInfo, ri *RetInfo) (err error) {
	if ci.chanRet == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	ri.cb = ci.cb
	ci.chanRet <- ri

	return
}

func (s *Server) exec(ci *CallInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if config.LenStackBuf > 0 {
				buf := make([]byte, config.LenStackBuf)
				l := runtime.Stack(buf, false)
				err = fmt.Errorf("%v:%s", r, buf[:l])
			} else {
				err = fmt.Errorf("%v", r)
			}
			s.ret(ci, &RetInfo{err: fmt.Errorf("%v", r)})
		}
	}()

	switch ci.f.(type) {
	case func([]interface{}):
		ci.f.(func([]interface{}))(ci.args)
		return s.ret(ci, &RetInfo{})
	case func([]interface{}) interface{}:
		ret := ci.f.(func([]interface{}) interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})
	case func([]interface{}) []interface{}:
		ret := ci.f.(func([]interface{}) []interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})
	}

	log.Log.Panic("bug")
	return
}

func (s *Server) Exec(ci *CallInfo) {
	err := s.exec(ci)
	if err != nil {
		log.Log.WithField("err", err).Error("call error")
	}
}

func (s *Server) Go(id interface{}, args ...interface{}) {
	fc := s.functions[id]
	if fc == nil {
		return
	}

	defer func() {
		recover()
	}()

	s.ChanCall <- &CallInfo{
		f:    fc,
		args: args,
	}
}

func (s *Server) Open(cLen int) *Client {
	client := NewClient(cLen)
	client.Attach(s)
	return client
}

// goroutine safe
func (s *Server) Call0(id interface{}, args ...interface{}) error {
	return s.Open(0).Call0(id, args...)
}

// goroutine safe
func (s *Server) Call1(id interface{}, args ...interface{}) (interface{}, error) {
	return s.Open(0).Call1(id, args...)
}

// goroutine safe
func (s *Server) CallN(id interface{}, args ...interface{}) ([]interface{}, error) {
	return s.Open(0).CallN(id, args...)
}

func (s *Server) Close() {
	close(s.ChanCall)

	for ci := range s.ChanCall {
		s.ret(ci, &RetInfo{
			err: errors.New("chanrpc server closed"),
		})
	}
}

func NewClient(cLen int) *Client {
	client := new(Client)
	client.chanSyncRet = make(chan *RetInfo, 1)
	client.ChanAsynRet = make(chan *RetInfo, cLen)

	return client
}

func (c *Client) Attach(server *Server) {
	c.s = server
}

func (c *Client) call(ci *CallInfo, block bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	if block {
		c.s.ChanCall <- ci
	} else {
		select {
		case c.s.ChanCall <- ci:
		default:
			err = errors.New("chanrpc channel full")
		}
	}

	return
}

func (c *Client) f(id interface{}, n int) (f interface{}, err error) {
	if c.s == nil {
		err = errors.New("server not attached")
		return
	}

	f = c.s.functions[id]

	if f == nil {
		err = fmt.Errorf("function id %v: function not registered", id)
		return
	}

	var ok bool
	switch n {
	case 0:
		_, ok = f.(func([]interface{}))
	case 1:
		_, ok = f.(func([]interface{}) interface{})
	case 2:
		_, ok = f.(func([]interface{}) []interface{})
	default:
		panic("bug")
	}

	if !ok {
		err = fmt.Errorf("function id %v: return type mismatch", id)
	}
	return
}

func (c *Client) Call0(id interface{}, args ...interface{}) error {
	f, err := c.f(id, 0)
	if err != nil {
		return err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)

	if err != nil {
		return err
	}

	ri := <-c.chanSyncRet

	return ri.err
}

func (c *Client) Call1(id interface{}, args ...interface{}) (interface{}, error) {
	f, err := c.f(id, 1)
	if err != nil {
		return nil, err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)

	if err != nil {
		return nil, err
	}

	ri := <-c.chanSyncRet

	return ri.ret, ri.err
}

func (c *Client) CallN(id interface{}, args ...interface{}) ([]interface{}, error) {
	f, err := c.f(id, 2)
	if err != nil {
		return nil, err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)

	if err != nil {
		return nil, err
	}

	ri := <-c.chanSyncRet

	return assert(ri.ret), ri.err
}

func (c *Client) asyncCall(id interface{}, args []interface{}, cb interface{}, n int) {
	f, err := c.f(id, n)
	if err != nil {
		c.ChanAsynRet <- &RetInfo{err: err, cb: cb}
		return
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.ChanAsynRet,
		cb:      cb,
	}, false)

	if err != nil {
		c.ChanAsynRet <- &RetInfo{err: err, cb: cb}
		return
	}
}

func (c *Client) AsynCall(id interface{}, _args ...interface{}) {
	if len(_args) < 1 {
		log.Log.Fatal("callback function not found")
	}

	args := _args[:len(_args)-1]
	cb := _args[len(_args)-1]

	var n int
	switch cb.(type) {
	case func(error):
		n = 0
	case func(interface{}, error):
		n = 1
	case func([]interface{}, error):
		n = 2
	default:
		log.Log.Fatal("definition of callback function is invalid")
	}

	if c.PendingAsynCall >= cap(c.ChanAsynRet) {
		execCb(&RetInfo{err: errors.New("too many calls"), cb: cb})
		return
	}

	c.asyncCall(id, args, cb, n)
	c.PendingAsynCall++
}

func execCb(ri *RetInfo) {
	defer func() {
		if r := recover(); r != nil {
			if config.LenStackBuf > 0 {
				buf := make([]byte, config.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Log.WithFields(logrus.Fields{"r": r, "buf": buf[:l]}).Error()
			} else {
				log.Log.WithField("r", r).Error()
			}
		}
	}()

	// execute
	switch ri.cb.(type) {
	case func(error):
		ri.cb.(func(error))(ri.err)
	case func(interface{}, error):
		ri.cb.(func(interface{}, error))(ri.ret, ri.err)
	case func([]interface{}, error):
		ri.cb.(func([]interface{}, error))(assert(ri.ret), ri.err)
	default:
		log.Log.Fatal("bug")
	}
}

func (c *Client) Cb(ri *RetInfo) {
	c.PendingAsynCall--
	execCb(ri)
}

func (c *Client) Close() {
	for c.PendingAsynCall > 0 {
		c.Cb(<-c.ChanAsynRet)
	}
}

func (c *Client) Idle() bool {
	return c.PendingAsynCall == 0
}
