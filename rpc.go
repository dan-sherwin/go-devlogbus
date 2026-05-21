package godevlogbus

import app_settings "github.com/dan-sherwin/go-app-settings"

type EmptyArgs struct{}

type EnabledArgs struct {
	Enabled bool
}

type EndpointArgs struct {
	Endpoint string
}

type ConfigureArgs struct {
	Enabled  bool
	Endpoint string
}

type RPCReceiver struct {
	Settings *Settings
	Handler  *Handler
	Persist  bool
}

func NewRPCReceiver(settings *Settings, handler *Handler) *RPCReceiver {
	return &RPCReceiver{Settings: settings, Handler: handler, Persist: true}
}

func (r *RPCReceiver) Status(_ EmptyArgs, reply *Status) error {
	*reply = r.status()
	return nil
}

func (r *RPCReceiver) Configure(args ConfigureArgs, reply *Status) error {
	if _, err := ParseEndpoint(args.Endpoint); err != nil {
		return err
	}
	if r.Persist {
		if err := app_settings.SetSetting(SettingEndpoint, args.Endpoint); err != nil {
			return err
		}
		if err := app_settings.SetSetting(SettingEnabled, args.Enabled); err != nil {
			return err
		}
	} else if err := r.configure(Config{Enabled: args.Enabled, Endpoint: args.Endpoint}); err != nil {
		return err
	}
	*reply = r.status()
	return nil
}

func (r *RPCReceiver) Enable(_ EmptyArgs, reply *Status) error {
	if r.Persist {
		if err := app_settings.SetSetting(SettingEnabled, true); err != nil {
			return err
		}
	} else if err := r.setEnabled(true); err != nil {
		return err
	}
	*reply = r.status()
	return nil
}

func (r *RPCReceiver) Disable(_ EmptyArgs, reply *Status) error {
	if r.Persist {
		if err := app_settings.SetSetting(SettingEnabled, false); err != nil {
			return err
		}
	} else if err := r.setEnabled(false); err != nil {
		return err
	}
	*reply = r.status()
	return nil
}

func (r *RPCReceiver) SetEnabled(args EnabledArgs, reply *Status) error {
	if r.Persist {
		if err := app_settings.SetSetting(SettingEnabled, args.Enabled); err != nil {
			return err
		}
	} else if err := r.setEnabled(args.Enabled); err != nil {
		return err
	}
	*reply = r.status()
	return nil
}

func (r *RPCReceiver) SetEndpoint(args EndpointArgs, reply *Status) error {
	if _, err := ParseEndpoint(args.Endpoint); err != nil {
		return err
	}
	if r.Persist {
		if err := app_settings.SetSetting(SettingEndpoint, args.Endpoint); err != nil {
			return err
		}
	} else if err := r.setEndpoint(args.Endpoint); err != nil {
		return err
	}
	*reply = r.status()
	return nil
}

func (r *RPCReceiver) configure(config Config) error {
	if r.Settings != nil {
		return r.Settings.Configure(config)
	}
	if r.Handler != nil {
		return r.Handler.Configure(config)
	}
	return nil
}

func (r *RPCReceiver) setEnabled(enabled bool) error {
	if r.Settings != nil {
		return r.Settings.SetEnabled(enabled)
	}
	status := r.status()
	return r.configure(Config{Enabled: enabled, Endpoint: status.Endpoint})
}

func (r *RPCReceiver) setEndpoint(endpoint string) error {
	if r.Settings != nil {
		return r.Settings.SetEndpoint(endpoint)
	}
	status := r.status()
	return r.configure(Config{Enabled: status.Enabled, Endpoint: endpoint})
}

func (r *RPCReceiver) status() Status {
	if r.Settings != nil {
		return r.Settings.Status()
	}
	if r.Handler != nil {
		return r.Handler.Status()
	}
	return Status{}
}
