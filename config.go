package godevlogbus

type config struct {
	Enabled  bool
	Endpoint string
}

type Status struct {
	Enabled    bool
	Endpoint   string
	Source     string
	Generation uint64
	LastError  string
}
