package config

var (
	LenStackBuf = 4096

	// log
	LogLevel string
	LogPath string
	LogFlag int

	// console
	ConsolePort int
	ConsolePrompt string = "Leaf#"
	ProfilePath string

	// cluster
	ListenAddr string
	ConnAddrs []string
	PendingWriterNum int
)