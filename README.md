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

Register these from the service `init()` path before `app_settings.Setup(...)`:

```go
var devLogBusSettings = godevlogbus.NewSettings()

func init() {
	devLogBusSettings.Register()
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

Attach the handler regardless of the enabled setting:

```go
handler := devLogBusSettings.NewHandler(godevlogbus.Options{
	Source: consts.APPNAME,
	Level:  level,
})
handlers = append(handlers, handler)
```

The handler drops records when disabled, and it drops records instead of
blocking when the broker cannot be reached or its internal queue is full.

## RPC

Register the receiver on the service's existing RPC server:

```go
handler := devLogBusSettings.NewHandler(godevlogbus.Options{
	Source: consts.APPNAME,
	Level:  level,
})

rpc.RegisterName(godevlogbus.DefaultRPCName, godevlogbus.NewRPCReceiver(devLogBusSettings, handler))
```

The receiver exposes:

- `DevLogBus.Status`
- `DevLogBus.Enable`
- `DevLogBus.Disable`
- `DevLogBus.SetEnabled`
- `DevLogBus.SetEndpoint`
- `DevLogBus.Configure`

By default the receiver persists changes through `app_settings.SetSetting`, so a
runtime troubleshooting change survives a service restart. Set `Persist=false`
on the receiver when a command should be process-local only.
