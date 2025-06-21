package request

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/buarki/caddy-nats-bridge/common"
	"github.com/buarki/caddy-nats-bridge/natsbridge"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var ErrTimeoutNotInitialized = errors.New("timeout not initialized")

type Request struct {
	Subject     string         `json:"subject,omitempty"`
	Timeout     *time.Duration `json:"timeout,omitempty"`
	ServerAlias string         `json:"serverAlias,omitempty"`

	logger *zap.Logger
	app    *natsbridge.NatsBridgeApp
}

func (Request) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.nats_request",
		New: func() caddy.Module {
			// Default values
			return &Request{
				ServerAlias: "default",
			}
		},
	}
}

func (p *Request) Provision(ctx caddy.Context) error {
	p.logger = ctx.Logger(p)

	natsAppIface, err := ctx.App("nats")
	if err != nil {
		return fmt.Errorf("getting NATS app: %v. Make sure NATS is configured in nats options", err)
	}

	p.app = natsAppIface.(*natsbridge.NatsBridgeApp)

	server, ok := p.app.Servers[p.ServerAlias]
	if !ok {
		return fmt.Errorf("NATS server alias %s not found", p.ServerAlias)
	}

	routeLevelTimeoutNotDefined := p.Timeout == nil
	if routeLevelTimeoutNotDefined {
		if server.DefaultTimeout != nil {
			p.Timeout = server.DefaultTimeout
			p.logger.Debug("using global default timeout", zap.Duration("timeout", *p.Timeout))
		} else {
			p.Timeout = &natsbridge.DefaultTimeout
			p.logger.Debug("using fallback default timeout", zap.Duration("timeout", *p.Timeout))
		}
	} else {
		p.logger.Debug("using custom timeout at route level", zap.Duration("timeout", *p.Timeout))
	}

	return nil
}

func (p Request) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	common.AddNATSPublishVarsToReplacer(repl, r)

	//TODO: What method is best here? ReplaceAll vs ReplaceWithErr?
	subj := repl.ReplaceAll(p.Subject, "")

	//p.logger.Debug("publishing NATS message", zap.String("subject", subj), zap.Bool("with_reply", p.WithReply), zap.Int64("timeout", p.Timeout))
	p.logger.Debug("publishing NATS message", zap.String("subject", subj))

	server, ok := p.app.Servers[p.ServerAlias]
	if !ok {
		return fmt.Errorf("NATS server alias %s not found", p.ServerAlias)
	}

	msg, err := common.NatsMsgForHttpRequest(r, subj)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		p.logger.Warn(fmt.Sprintf("Request sent with invalid characters %v", err.Error()))
		return nil
	}

	start := time.Now()
	defer func() {
		p.logger.Debug("http_request", zap.String("duration", fmt.Sprintf("%d ms", time.Since(start).Milliseconds())))
	}()

	if p.Timeout == nil {
		p.logger.Error("timeout not initialized", zap.String("subject", subj))
		return ErrTimeoutNotInitialized
	}

	resp, err := server.Conn.RequestMsg(msg, *p.Timeout)
	if err != nil && errors.Is(err, nats.ErrNoResponders) {
		w.WriteHeader(http.StatusNotFound)
		p.logger.Warn("No Responders for NATS subject - answering with HTTP Status Not Found.", zap.String("subject", subj), zap.String("timeout", p.Timeout.String()))
		return nil
	}
	p.logger.Debug("nats_request", zap.String("duration", fmt.Sprintf("%d ms", time.Since(start).Milliseconds())))
	if err != nil && errors.Is(err, nats.ErrTimeout) {
		w.WriteHeader(http.StatusGatewayTimeout)
		p.logger.Warn("Request timed out", zap.String("subject", subj), zap.String("timeout", p.Timeout.String()))
		return nil
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("could not request NATS message: %w", err)
	}

	for k, headers := range resp.Header {
		// strip out these headers from the response
		if k == "Nats-Service-Error" || k == "Nats-Service-Error-Code" || k == "nats-service-error" || k == "nats-service-error-code" || k == "Content-Length" {
			continue
		}
		for _, header := range headers {
			w.Header().Add(k, header)
		}
	}

	code := resp.Header.Get("Nats-Service-Error-Code")
	if code != "" && code != "200" {
		status, err := strconv.Atoi(code)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("error converting status: %w", err)
		}
		w.WriteHeader(status)
	}

	_, err = w.Write(resp.Data)
	if err != nil {
		return fmt.Errorf("could not write response back to HTTP Writer: %w", err)
	}

	// we are done :)
	return nil
}

var (
	_ caddyhttp.MiddlewareHandler = (*Request)(nil)
	_ caddy.Provisioner           = (*Request)(nil)
	_ caddyfile.Unmarshaler       = (*Request)(nil)
)
