package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/startstop"
)

var _ startstop.Starter = (*Server)(nil)

type Server struct {
	log      *slog.Logger
	config   *Config
	servers  []*http.Server
	launched bool
}

type Config struct {
	Handler    http.Handler
	ACMEHosts  []string
	Port       int
	EnableACME bool
}

// New creates a new HTTP server using the provided config.
func New(config *Config) *Server {
	debug.Assert(config.Handler != nil, "expected Handler to be configured")

	return &Server{
		log:      slog.Default().With("name", "Server"),
		config:   config,
		launched: false,
		servers:  make([]*http.Server, 0, 2),
	}
}

func (s *Server) Start(ctx context.Context) error {
	if s.launched {
		return errors.New("common/http: server already launched")
	}

	s.launched = true

	if s.config.EnableACME {
		return s.listenAndServeTLS(ctx)
	}

	return s.listenAndServe(ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	deadline, _ := ctx.Deadline()
	s.log.InfoContext(ctx, "Gracefully shutting down server", "deadline", deadline)

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	go func() {
		<-shutdownCtx.Done()

		if shutdownCtx.Err() == context.DeadlineExceeded {
			s.log.ErrorContext(shutdownCtx, "Server shutdown timed out")
		}
	}()

	var g errgroup.Group

	for _, srv := range s.servers {
		csrv := srv

		g.Go(func() error {
			return csrv.Shutdown(shutdownCtx)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("httpserver: failed to stop servers: %w", err)
	}

	return nil
}

func (s *Server) listenAndServe(ctx context.Context) error {
	srv := &http.Server{Handler: s.config.Handler}

	l, err := listener(s.config.Port)
	if err != nil {
		return err
	}

	s.servers = append(s.servers, srv)

	return s.serve(ctx, l, srv)
}

func (s *Server) listenAndServeTLS(ctx context.Context) error {
	var g errgroup.Group

	//nolint:exhaustruct
	m := &autocert.Manager{
		Cache:      autocert.DirCache(".cache"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.config.ACMEHosts...),
	}

	baseContext := func(net.Listener) context.Context {
		// TODO(roman@vanesyan.com): use this to accept shutdown context instead.
		return context.Background()
	}

	srv80 := &http.Server{
		Handler:     m.HTTPHandler(nil),
		BaseContext: baseContext,
	}
	srv443 := &http.Server{
		Handler:     s.config.Handler,
		BaseContext: baseContext,
	}

	g.Go(func() error {
		l, err := listener(80)
		if err != nil {
			return err
		}

		return s.serve(ctx, l, srv80)
	})

	g.Go(func() error {
		l, err := listenerTLS(443, m)
		if err != nil {
			return err
		}

		return s.serve(ctx, l, srv443)
	})

	s.servers = append(s.servers, srv80, srv443)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("httpserver: failed to start servers: %w", err)
	}

	return nil
}

func (s *Server) serve(ctx context.Context, l net.Listener, srv *http.Server) error {
	s.log.InfoContext(ctx, "Starting server", "addr", l.Addr().String())

	err := srv.Serve(l)
	if err != nil && err != http.ErrServerClosed {
		s.log.ErrorContext(ctx, "Server failed to start", "error", err)
		return fmt.Errorf("httpserver: failed to start server on %s: %w", l.Addr().String(), err)
	}

	return nil
}

func listener(port int) (net.Listener, error) {
	listener, err := net.ListenTCP(
		"tcp",
		//nolint:exhaustruct
		&net.TCPAddr{Port: port})
	if err != nil {
		return nil, fmt.Errorf("httpserver: failed to start listener on port %d: %w", port, err)
	}

	return listener, nil
}

func listenerTLS(port int, m *autocert.Manager) (net.Listener, error) {
	l, err := listener(port)
	if err != nil {
		return nil, err
	}

	//nolint:exhaustruct
	l = tls.NewListener(l, &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
	})

	return l, nil
}
