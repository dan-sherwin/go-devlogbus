package godevlogbus

import (
	"fmt"
	"io"
)

type (
	CommandDef struct {
		DevLogBus command `cmd:"" name:"devlogbus" help:"Manage runtime DevLogBus logging"`
	}
	command struct {
		Status      statusCommand      `cmd:"" help:"Show runtime DevLogBus status" default:"1"`
		Enable      enableCommand      `cmd:"" help:"Enable runtime DevLogBus publishing"`
		Disable     disableCommand     `cmd:"" help:"Disable runtime DevLogBus publishing"`
		SetEndpoint setEndpointCommand `cmd:"" name:"setEndpoint" help:"Set the runtime DevLogBus endpoint"`
	}
	statusCommand struct{}
	enableCommand struct {
		Endpoint string `help:"Optional endpoint to set before enabling"`
	}
	disableCommand     struct{}
	setEndpointCommand struct {
		Endpoint string `arg:"" help:"Unix socket path, unix:/path.sock, tcp://host:port, or host:port" required:""`
	}
)

func (c *statusCommand) Run() error {
	status, err := currentStatus()
	if err != nil {
		return err
	}
	printStatus(runtimeWriter(), status)
	return nil
}

func (c *enableCommand) Run() error {
	status, err := enable(c.Endpoint)
	if err != nil {
		return err
	}
	printStatus(runtimeWriter(), status)
	return nil
}

func (c *disableCommand) Run() error {
	status, err := disable()
	if err != nil {
		return err
	}
	printStatus(runtimeWriter(), status)
	return nil
}

func (c *setEndpointCommand) Run() error {
	status, err := setEndpoint(c.Endpoint)
	if err != nil {
		return err
	}
	printStatus(runtimeWriter(), status)
	return nil
}

func printStatus(writer io.Writer, status Status) {
	if writer == nil {
		writer = io.Discard
	}
	fmt.Fprintf(writer, "Enabled:    %t\n", status.Enabled)
	fmt.Fprintf(writer, "Endpoint:   %s\n", status.Endpoint)
	fmt.Fprintf(writer, "Source:     %s\n", status.Source)
	fmt.Fprintf(writer, "Generation: %d\n", status.Generation)
	if status.LastError != "" {
		fmt.Fprintf(writer, "Last Error: %s\n", status.LastError)
	}
}
