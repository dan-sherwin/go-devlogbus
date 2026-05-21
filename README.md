# GoDevLogBus

Spacelink-local runtime glue for wiring DevLogBus into the templated Go
services that use `app_settings` and the built-in Unix RPC listener.

This package intentionally lives outside `github.com/dan-sherwin/devlogbus`.
DevLogBus stays a general transport/broker project; this package owns the
Spacelink templated-service runtime behavior:

- always attach a `slog.Handler`
- keep the handler inert when `devlogbus_enabled=false`
- let `devlogbus_endpoint` change at runtime
- accept Unix socket paths and TCP endpoints
- expose an RPC receiver that running services can register
- persist RPC changes back through `app_settings` when desired

## Settings

Most templated services should use `Runtime`. It owns the settings, RPC
receiver, handler lifecycle, and CLI command implementation:

```go
var devLogBus = godevlogbus.NewRuntime(godevlogbus.RuntimeOptions{
	Source:      consts.APPNAME,
	RegisterRPC: rpc.RegisterName,
	CallRPC:     rpc.Call,
})

func init() {
	devLogBus.Register()
}

func withDevLogBusHandler(handlers []slog.Handler, level slog.Level) []slog.Handler {
	return devLogBus.WithHandler(handlers, level)
}
```

The canonical settings are:

- `devlogbus_enabled`
- `devlogbus_endpoint`

`devlogbus_socket_path` is registered as a compatibility bridge by default. New
services should use `devlogbus_endpoint`.

Endpoint examples:

- `/tmp/devlogbus/devlogbus.sock`
- `unix:/tmp/devlogbus/devlogbus.sock`
- `tcp://127.0.0.1:7422`
- `prod-debug-host:7422`

## Handler

Attach the runtime handler regardless of the enabled setting:

```go
handlers = devLogBus.WithHandler(handlers, level)
```

The handler drops records when disabled, and it drops records instead of
blocking when the broker cannot be reached or its internal queue is full.

## CLI And RPC

Embed the package command definition in the service's Kong command tree:

```go
type Commands struct {
	godevlogbus.CommandDef
}
```

The receiver exposes:

- `DevLogBus.Status`
- `DevLogBus.Enable`
- `DevLogBus.Disable`
- `DevLogBus.SetEnabled`
- `DevLogBus.SetEndpoint`
- `DevLogBus.Configure`

The CLI command exposes:

- `devlogbus status`
- `devlogbus enable`
- `devlogbus disable`
- `devlogbus setEndpoint`

By default the receiver persists changes through `app_settings.SetSetting`, so a
runtime troubleshooting change survives a service restart. Set `Persist=false`
on `RPCReceiver`, or `DisableRPCPersistence=true` on `RuntimeOptions`, when a
command should be process-local only.
