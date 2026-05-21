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

## Usage

Most templated services should call `Setup` once from the service `init()` path.
The package owns a single process-wide runtime for settings, RPC, handler
lifecycle, and CLI command implementation:

```go
func init() {
	godevlogbus.Setup(godevlogbus.SetupOptions{
		Source:      consts.APPNAME,
		RegisterRPC: rpc.RegisterName,
		CallRPC:     rpc.Call,
	})
}
```

The logger can attach the handler directly from the package:

```go
handlers = godevlogbus.WithHandler(handlers, level)
```

The canonical settings are:

- `devlogbus_enabled`
- `devlogbus_endpoint`

Endpoint examples:

- `/tmp/devlogbus/devlogbus.sock`
- `unix:/tmp/devlogbus/devlogbus.sock`
- `tcp://127.0.0.1:7422`
- `prod-debug-host:7422`

## Handler

Attach the runtime handler regardless of the enabled setting:

```go
handlers = godevlogbus.WithHandler(handlers, level)
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
- `DevLogBus.SetEndpoint`
- `DevLogBus.Configure`

The CLI command exposes:

- `devlogbus status`
- `devlogbus enable`
- `devlogbus disable`
- `devlogbus setEndpoint`

Status output is intentionally terse:

```text
Enabled:    true
Endpoint:   tcp://127.0.0.1:7422
Source:     event_management_svc
Generation: 2
```

By default the receiver persists changes through `app_settings.SetSetting`, so a
runtime troubleshooting change survives a service restart. Set
`DisableRPCPersistence=true` on `SetupOptions` when a command should be
process-local only.
