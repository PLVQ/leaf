package network

type Processor interface {
	Route(msg interface{}, userData interface{}) error
	Unmarshal(data []byte) (interface{}, error)
	Marshal(msg interface{}) ([][]byte, error)
}
