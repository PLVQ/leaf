package network

type Agent interface {
	Run()
	Close()
}
