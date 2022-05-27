package chanrpc

import (
	"github.com/jiangzuomin/leaf/log"
	"sync"
	"testing"
)

func TestChanrpc(t *testing.T) {
	s := NewServer(10)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		s.Register("f0", func(args []interface{}) {
			log.Log.Info("call f0")
		})

		s.Register("f1", func(args []interface{}) interface{} {
			log.Log.Info("call f1")
			return 1
		})

		s.Register("f2", func(args []interface{}) []interface{} {
			return []interface{}{args[0], args[1]}
		})

		s.Register("add", func(args []interface{}) interface{} {
			if len(args) < 2 {
				log.Log.Error("args error")
				return -1
			}

			return args[0].(int) + args[1].(int)
		})

		wg.Done()

		for {
			s.Exec(<-s.ChanCall)
		}
	}()

	wg.Wait()

	wg.Add(1)

	go func() {
		c := s.Open(10)

		err := c.Call0("f0")
		if err != nil {
			log.Log.WithField("err", err).Error()
		}

		ret, err := c.Call1("f1")
		if err != nil {
			log.Log.WithField("err", err).Error()
		} else {
			log.Log.WithField("ret", ret).Info()
		}

		ret, err = c.CallN("f2", 1, 2)
		if err != nil {
			log.Log.WithField("err", err).Error()
		} else {
			log.Log.WithField("ret", ret).Info()
		}

		ret, err = c.Call1("add", 1, 2)
		if err != nil {
			log.Log.WithField("err", err).Error()
		} else {
			log.Log.WithField("ret", ret).Info()
		}

		c.AsyncCall("f0", func(err error) {
			if err != nil {
				log.Log.WithField("err", err).Error()
			} else {
				log.Log.Info("asyncCall f0")
			}
		})

		c.AsyncCall("f1", func(ret interface{}, err error) {
			if err != nil {
				log.Log.WithField("err", err).Error()
			} else {
				log.Log.WithField("ret", ret).Info()
			}
		})

		c.AsyncCall("f2", 1, 2, func(ret []interface{}, err error) {
			if err != nil {
				log.Log.WithField("err", err).Error()
			} else {
				log.Log.WithField("ret", ret).Info()
			}
		})

		c.AsyncCall("add", 1, 2, func(ret interface{}, err error) {
			if err != nil {
				log.Log.WithField("err", err).Error()
			} else {
				log.Log.WithField("ret", ret).Info()
			}
		})

		c.Cb(<-c.ChanAsynRet)
		c.Cb(<-c.ChanAsynRet)
		c.Cb(<-c.ChanAsynRet)
		c.Cb(<-c.ChanAsynRet)

		wg.Done()
	}()

	wg.Wait()
}
