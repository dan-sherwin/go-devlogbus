package godevlogbus

import (
	"fmt"
	"io"
)

type (
	CommandDef struct {
		DevLogBus Command `cmd:"" name:"devlogbus" help:"Manage runtime DevLogBus logging"`
	}
	Command struct {
		Status      StatusCommand      `cmd:"" help:"Show runtime DevLogBus status" default:"1"`
		Enable      EnableCommand      `cmd:"" help:"Enable runtime DevLogBus publishing"`
		Disable     DisableCommand     `cmd:"" help:"Disable runtime DevLogBus publishing"`
		SetEndpoint SetEndpointCommand `cmd:"" name:"setEndpoint" help:"Set the runtime DevLogBus endpoint"`
	}
	StatusCommand struct{}
	EnableCommand struct {
		Endpoint string `help:"Optional endpoint to set before enabling"`
	}
	DisableCommand     struct{}
	SetEndpointCommand struct {
		Endpoint string `arg:"" help:"Unix socket path, unix:/path.sock, tcp://host:port, or host:port" required:""`
	}
)

func (c *StatusCommand) Run() error {
	status, err := CurrentStatus()
	if err != nil {
		return err
	}
	FprintStatus(runtimeWriter(), status)
	return nil
}

func (c *EnableCommand) Run() error {
	status, err := Enable(c.Endpoint)
	if err != nil {
		return err
	}
	FprintStatus(runtimeWriter(), status)
	return nil
}

func (c *DisableCommand) Run() error {
	status, err := Disable()
	if err != nil {
		return err
	}
	FprintStatus(runtimeWriter(), status)
	return nil
}

func (c *SetEndpointCommand) Run() error {
	status, err := SetEndpoint(c.Endpoint)
	if err != nil {
		return err
	}
	FprintStatus(runtimeWriter(), status)
	return nil
}

func FprintStatus(writer io.Writer, status Status) {
	if writer == nil {
		writer = io.Discard
	}
	fmt.Fprintf(writer, "Enabled:         %t\n", status.Enabled)
	fmt.Fprintf(writer, "Endpoint:        %s\n", status.Endpoint)
	fmt.Fprintf(writer, "Network:         %s\n", status.Network)
	fmt.Fprintf(writer, "Address:         %s\n", status.Address)
	if status.SocketPath != "" {
		fmt.Fprintf(writer, "Socket Path:     %s\n", status.SocketPath)
	}
	fmt.Fprintf(writer, "Source:          %s\n", status.Source)
	fmt.Fprintf(writer, "Generation:      %d\n", status.Generation)
	if status.LastError != "" {
		fmt.Fprintf(writer, "Last Error:      %s\n", status.LastError)
	}
}
