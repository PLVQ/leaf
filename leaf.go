package leaf

import (
	"github.com/jiangzuomin/leaf/cluster"
	"github.com/jiangzuomin/leaf/console"
	"github.com/jiangzuomin/leaf/log"
	"github.com/jiangzuomin/leaf/module"
	"os"
	"os/signal"
)

func Run(mods ...module.Module) {
	log.Log.Info("Leaf starting up")

	// module
	for i := 0; i < len(mods); i++ {
		module.Register(mods[i])
	}
	module.Init()

	// cluster
	cluster.Init()

	// console
	console.Init()

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Log.WithField("signal", sig).Error("Leaf closing down")
	console.Destroy()
	cluster.Destroy()
	module.Destroy()
}
