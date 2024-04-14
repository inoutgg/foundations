package server

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"log/slog"

	"github.com/atcirclesquare/common/startstop"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

var _ startstop.Starter = (*Server)(nil)

type Server struct {
	log      *slog.Logger
	config   *ServerConfig
	servers  []*http.Server
	launched bool
}

type ServerConfig struct {
	Handler http.Handler

	// If EnableACME flag is set Port option is ignored.
	Port int

	// If EnableACME is true, then the server will use Let's Encrypt to automatically.
	EnableACME bool
	ACMEHosts  []string
}

func NewServer(config *ServerConfig) *Server {
	return &Server{
		log:    slog.Default().With("name", "Server"),
		config: config,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if s.launched {
		return errors.New("common/http: server already launched")
	}

	s.launched = true

	if s.config.EnableACME {
		return s.listenAndServeTLS()
	}

	return s.listenAndServe()
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

	return g.Wait()
}

func (s *Server) listenAndServe() error {
	srv := &http.Server{Handler: s.config.Handler}

	l, err := listener(s.config.Port)
	if err != nil {
		return err
	}

	s.servers = append(s.servers, srv)

	return s.serve(l, srv)
}

func (s *Server) listenAndServeTLS() error {
	var g errgroup.Group
	m := &autocert.Manager{
		Cache:      autocert.DirCache(".cache"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.config.ACMEHosts...),
	}

	srv1 := &http.Server{Handler: m.HTTPHandler(nil)}
	srv2 := &http.Server{Handler: s.config.Handler}

	g.Go(func() error {
		l, err := listener(80)
		if err != nil {
			return err
		}

		return s.serve(l, srv1)
	})

	g.Go(func() error {
		l, err := listenerTLS(443, m)
		if err != nil {
			return err
		}

		return s.serve(l, srv2)
	})

	s.servers = append(s.servers, srv1, srv2)

	return g.Wait()
}

func (s *Server) serve(l net.Listener, srv *http.Server) error {
	s.log.Info("Starting server", "addr", l.Addr().String())
	err := srv.Serve(l)
	if err != nil && err != http.ErrServerClosed {
		s.log.Error("Server failed to start", "error", err)
		return err
	}

	return nil
}

func listener(port int) (net.Listener, error) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func listenerTLS(port int, m *autocert.Manager) (net.Listener, error) {
	l, err := listener(port)
	if err != nil {
		return nil, err
	}

	l = tls.NewListener(l, &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: m.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
	})

	return l, nil
}
