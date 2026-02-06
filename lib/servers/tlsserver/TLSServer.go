// Package tlsserver with TLS server for use by plugins and testing
package tlsserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

// TLSServer is a simple TLS MsgServer supporting BASIC, Jwt and client certificate authentication
type TLSServer struct {
	address    string
	port       int
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	httpServer *http.Server
	router     *chi.Mux
}

// Router returns the router for this server.
// This router has cors protection enabled.
func (srv *TLSServer) Router() *chi.Mux {
	return srv.router
}

// Start the TLS server using the provided CA and Server certificates.
// If a client certificate is provided it must be valid and signed by the CA.
// This configures handling of CORS requests to allow:
//   - any origin by returning the requested origin (not using wildcard '*').
//   - any method, eg PUT, POST, GET, PATCH,
//   - headers "Origin", "Accept", "Content-Type", "X-Requested-With"
func (srv *TLSServer) Start() error {
	var err error

	slog.Info("Starting TLS server", "address", srv.address, "port", srv.port)
	if srv.caCert == nil || srv.serverCert == nil {
		err := fmt.Errorf("missing CA or server certificate")
		slog.Error(err.Error())
		return err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(srv.caCert)

	serverTLSConf := &tls.Config{
		Certificates:       []tls.Certificate{*srv.serverCert},
		ClientAuth:         tls.VerifyClientCertIfGiven,
		ClientCAs:          caCertPool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}
	logHandler := tint.NewHandler(os.Stdout, &tint.Options{
		AddSource: true, Level: slog.LevelInfo, TimeFormat: "Jan _2 15:04:05.0000"})
	logger := slog.NewLogLogger(logHandler, slog.LevelDebug)

	// handle CORS using the cors plugin
	// see also: https://stackoverflow.com/questions/43871637/no-access-control-allow-origin-header-is-present-on-the-requested-resource-whe
	// TODO: add configuration for CORS origin: allowed, sameaddress, exact
	c := cors.New(cors.Options{
		// return the origin as allowed origin
		AllowOriginFunc: func(orig string) bool {
			// local requests are always allowed, even over http (for testing) - todo: disable in production
			if strings.HasPrefix(orig, "https://127.0.0.1") || strings.HasPrefix(orig, "https://localhost") ||
				strings.HasPrefix(orig, "http://127.0.0.1") || strings.HasPrefix(orig, "http://localhost") {
				slog.Debug("TLSServer.AllowOriginFunc: Cors origin Is True", "origin", orig)
				return true
			} else if strings.HasPrefix(orig, "https://"+srv.address) {
				slog.Debug("TLSServer.AllowOriginFunc: Cors origin Is True", "origin", orig)
				return true
			} else if orig == "" {
				// same-origin is allowed
				return true
			}
			slog.Warn("TLSServer.AllowOriginFunc: Cors: invalid origin:", "origin", orig)
			// for testing just warn about invalid origin
			return true
		},
		// default allowed headers is "Origin", "Accept", "Content-Type", "X-Requested-With" (missing authorization)
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type", "Authorization", "Headers"},
		//AllowedHeaders: []string{"*"},
		// default is get/put/patch/post/delete/head
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		Debug:          false,
		//Debug:            true, // the AllowOriginFunc above does the reporting
		AllowCredentials: true,
		Logger:           logger,
	})
	handler := c.Handler(srv.router)

	srv.httpServer = &http.Server{
		Addr: fmt.Sprintf("%s:%d", srv.address, srv.port),
		// ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when 'curl' on OSx prompts for username/password
		// WriteTimeout: 10 * time.Second,
		Handler:   handler,
		TLSConfig: serverTLSConf,
		//ErrorLog:  log.Default(),
	}
	l, err := net.Listen("tcp", srv.httpServer.Addr)
	if err != nil {
		return err
	}
	// mutex to capture error result in case startup in the background failed
	go func() {
		// serverTLSConf contains certificate and key
		err2 := srv.httpServer.ServeTLS(l, "", "")
		//err2 := srv.httpServer.ListenAndServeTLS("", "")
		if err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			err = fmt.Errorf("TLS Server start error: %s", err2.Error())
			slog.Error(err.Error())
		} else {
			slog.Info("TLSServer stopped")
		}
	}()
	return err
}

// Stop the TLS server and close all connections
// this waits until for up to 3 seconds for connections are closed. After that
// continue.
func (srv *TLSServer) Stop() {
	slog.Info("Stopping TLS server")

	if srv.httpServer != nil {
		// note that this does not (cannot?) close existing client connections
		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*3)
		err := srv.httpServer.Shutdown(ctx)
		if err != nil {
			slog.Error("Stop: TLS server graceful shutdown failed. Forcing Remove", "err", err.Error())
			_ = srv.httpServer.Close()
		}
		cancelFn()
	}
}

// NewTLSServer creates a new TLS MsgServer instance with authentication support.
// This returns the chi-go router which can be used to add routes and middleware.
// This server supports the "message-id" header for received requests.
//
// Use Start() to start listening.
//
// The middleware handlers included with the server can be used for authentication.
//
//	address        server listening address
//	port           listening port
//	serverCert     Server TLS certificate
//	caCert         CA certificate to verify client certificates
//
// returns TLS server and router for handling requests
func NewTLSServer(address string, port int,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
) (*TLSServer, *chi.Mux) {

	srv := &TLSServer{
		caCert:     caCert,
		serverCert: serverCert,
		router:     chi.NewRouter(),
	}
	//// support for CORS response headers
	//srv.router.Use(mux.CORSMethodMiddleware(srv.router))

	srv.address = address
	srv.port = port
	return srv, srv.router
}
