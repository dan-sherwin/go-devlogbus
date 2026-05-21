package godevlogbus

type Config struct {
	Enabled  bool
	Endpoint string
}

type Status struct {
	Enabled        bool
	Endpoint       string
	Network        string
	Address        string
	SocketPath     string
	Source         string
	QueueSize      int
	PublishTimeout string
	Generation     uint64
	LastError      string
}
