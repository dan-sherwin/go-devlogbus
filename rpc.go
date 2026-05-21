package godevlogbus

import (
	"errors"

	app_settings "github.com/dan-sherwin/go-app-settings"
)

type EmptyArgs struct{}

type EndpointArgs struct {
	Endpoint string
}

type ConfigureArgs struct {
	Enabled  bool
	Endpoint string
}

type rpcReceiver struct {
	settings *settings
	persist  bool
}

func newRPCReceiver(settings *settings, persist bool) *rpcReceiver {
	return &rpcReceiver{settings: settings, persist: persist}
}

func (r *rpcReceiver) Status(_ EmptyArgs, reply *Status) error {
	status, err := r.status()
	if err != nil {
		return err
	}
	*reply = status
	return nil
}

func (r *rpcReceiver) Configure(args ConfigureArgs, reply *Status) error {
	if _, err := parseEndpoint(args.Endpoint); err != nil {
		return err
	}
	if r.persist {
		if args.Enabled {
			if err := app_settings.SetSetting(settingEndpoint, args.Endpoint); err != nil {
				return err
			}
			if err := app_settings.SetSetting(settingEnabled, true); err != nil {
				return err
			}
		} else {
			if err := app_settings.SetSetting(settingEnabled, false); err != nil {
				return err
			}
			if err := app_settings.SetSetting(settingEndpoint, args.Endpoint); err != nil {
				return err
			}
		}
	} else if err := r.settings.configure(config{Enabled: args.Enabled, Endpoint: args.Endpoint}); err != nil {
		return err
	}
	return r.Status(EmptyArgs{}, reply)
}

func (r *rpcReceiver) Enable(_ EmptyArgs, reply *Status) error {
	if r.persist {
		if err := app_settings.SetSetting(settingEnabled, true); err != nil {
			return err
		}
	} else if err := r.settings.setEnabled(true); err != nil {
		return err
	}
	return r.Status(EmptyArgs{}, reply)
}

func (r *rpcReceiver) Disable(_ EmptyArgs, reply *Status) error {
	if r.persist {
		if err := app_settings.SetSetting(settingEnabled, false); err != nil {
			return err
		}
	} else if err := r.settings.setEnabled(false); err != nil {
		return err
	}
	return r.Status(EmptyArgs{}, reply)
}

func (r *rpcReceiver) SetEndpoint(args EndpointArgs, reply *Status) error {
	if _, err := parseEndpoint(args.Endpoint); err != nil {
		return err
	}
	if r.persist {
		if err := app_settings.SetSetting(settingEndpoint, args.Endpoint); err != nil {
			return err
		}
	} else if err := r.settings.setEndpoint(args.Endpoint); err != nil {
		return err
	}
	return r.Status(EmptyArgs{}, reply)
}

func (r *rpcReceiver) status() (Status, error) {
	if r == nil || r.settings == nil {
		return Status{}, errors.New("devlogbus settings are not configured")
	}
	return r.settings.status(), nil
}
