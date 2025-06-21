package natsbridge

import (
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
)

func TestNatsBridgeApp_DefaultTimeout(t *testing.T) {
	tests := []struct {
		name            string
		setupServer     func(*NatsServer)
		serverAlias     string
		expectedTimeout time.Duration
		expectNil       bool
	}{
		{
			name: "default timeout not set - should use package default",
			setupServer: func(server *NatsServer) {
				// don't set DefaultTimeout - should use package default
			},
			serverAlias:     "default",
			expectedTimeout: DefaultTimeout,
			expectNil:       false,
		},
		{
			name: "default timeout set to custom value",
			setupServer: func(server *NatsServer) {
				customTimeout := 5 * time.Second
				server.DefaultTimeout = &customTimeout
			},
			serverAlias:     "default",
			expectedTimeout: 5 * time.Second,
			expectNil:       false,
		},
		{
			name: "default timeout set to zero - should keep zero value",
			setupServer: func(server *NatsServer) {
				zeroTimeout := time.Duration(0)
				server.DefaultTimeout = &zeroTimeout
			},
			serverAlias:     "default",
			expectedTimeout: 0,
			expectNil:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &NatsBridgeApp{
				Servers: make(map[string]*NatsServer),
			}

			server := &NatsServer{
				NatsUrl: "127.0.0.1:4222",
			}
			app.Servers[tt.serverAlias] = server

			if tt.setupServer != nil {
				tt.setupServer(server)
			}

			ctx := caddy.Context{}
			err := app.Provision(ctx)
			if err != nil {
				t.Fatalf("failed to provision app: %v", err)
			}

			server, exists := app.Servers[tt.serverAlias]
			if !exists {
				if !tt.expectNil {
					t.Errorf("expected server '%s' to exist, but it doesn't", tt.serverAlias)
				}
				return
			}

			if tt.expectNil {
				t.Errorf("expected server '%s' to not exist, but it does", tt.serverAlias)
				return
			}

			if server.DefaultTimeout == nil {
				t.Errorf("expected DefaultTimeout to be set, but it's nil")
				return
			}

			actualTimeout := *server.DefaultTimeout
			if actualTimeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, actualTimeout)
			}
		})
	}
}

func TestNatsBridgeApp_DefaultTimeout_MultipleServers(t *testing.T) {
	app := &NatsBridgeApp{
		Servers: make(map[string]*NatsServer),
	}

	server1 := &NatsServer{
		NatsUrl: "127.0.0.1:4222",
	}
	app.Servers["server1"] = server1

	customTimeout := 10 * time.Second
	server2 := &NatsServer{
		NatsUrl:        "127.0.0.1:4223",
		DefaultTimeout: &customTimeout,
	}
	app.Servers["server2"] = server2

	ctx := caddy.Context{}
	err := app.Provision(ctx)
	if err != nil {
		t.Fatalf("failed to provision app: %v", err)
	}

	if server1.DefaultTimeout == nil {
		t.Errorf("expected server1 DefaultTimeout to be set, but it's nil")
	} else if *server1.DefaultTimeout != DefaultTimeout {
		t.Errorf("expected server1 timeout to be %v, got %v", DefaultTimeout, *server1.DefaultTimeout)
	}

	if server2.DefaultTimeout == nil {
		t.Errorf("expected server2 DefaultTimeout to be set, but it's nil")
	} else if *server2.DefaultTimeout != customTimeout {
		t.Errorf("expected server2 timeout to be %v, got %v", customTimeout, *server2.DefaultTimeout)
	}
}

func TestNatsBridgeApp_DefaultTimeout_ProvisionOrder(t *testing.T) {
	app := &NatsBridgeApp{
		Servers: make(map[string]*NatsServer),
	}

	server := &NatsServer{
		NatsUrl: "127.0.0.1:4222",
	}
	app.Servers["default"] = server

	if server.DefaultTimeout != nil {
		t.Errorf("expected DefaultTimeout to be nil before provisioning, got %v", *server.DefaultTimeout)
	}

	ctx := caddy.Context{}
	err := app.Provision(ctx)
	if err != nil {
		t.Fatalf("failed to provision app: %v", err)
	}

	if server.DefaultTimeout == nil {
		t.Errorf("expected DefaultTimeout to be set after provisioning, but it's nil")
		return
	}

	if *server.DefaultTimeout != DefaultTimeout {
		t.Errorf("expected DefaultTimeout to be %v after provisioning, got %v",
			DefaultTimeout, *server.DefaultTimeout)
	}
}
