package godevlogbus

import (
	"net/rpc"
	"testing"
)

func TestRPCReceiverRegistersWithNetRPC(t *testing.T) {
	server := rpc.NewServer()
	if err := server.RegisterName(defaultRPCName, newRPCReceiver(newSettings(), false)); err != nil {
		t.Fatalf("register receiver: %v", err)
	}
}
