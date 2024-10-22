package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/services/idprov/idprovapi"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log/slog"
	"net/http"
)

// IdProvHttpServer serves the provisioning requests
type IdProvHttpServer struct {
	tlsServer *tlsserver.TLSServer
	mng       *ManageIdProvService
}

// Stop the http server
func (srv *IdProvHttpServer) Stop() {
	if srv.tlsServer != nil {
		srv.tlsServer.Stop()
		srv.tlsServer = nil
	}
}

func (srv *IdProvHttpServer) handleRequest(w http.ResponseWriter, req *http.Request) {
	slog.Info("handleRequest", slog.String("remoteAddr", req.RemoteAddr))

	args := idprovapi.ProvisionRequestArgs{}
	data, err := io.ReadAll(req.Body)
	err2 := jsoniter.Unmarshal(data, &args)
	if err != nil || err2 != nil {
		slog.Warn("idprov handleRequest. bad request", "remoteAddr", req.RemoteAddr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := srv.mng.SubmitRequest("unknown", &args)
	if err != nil {
		slog.Warn("idprov handleRequest. refused", "err", err.Error())
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	respData, _ := json.Marshal(resp)
	_, err = w.Write(respData)
	if err != nil {
		slog.Error("error sending response", "err", err.Error(), "remoteAddr", req.RemoteAddr)
	}
}

// StartIdProvHttpServer starts the http server to handle provisioning requests
func StartIdProvHttpServer(
	port int, serverCert *tls.Certificate, caCert *x509.Certificate, mng *ManageIdProvService) (*IdProvHttpServer, error) {

	tlsServer, mux := tlsserver.NewTLSServer("", port, serverCert, caCert)
	_ = mux
	srv := IdProvHttpServer{
		tlsServer: tlsServer,
		mng:       mng,
	}
	mux.Post(idprovapi.ProvisionRequestPath, srv.handleRequest)
	err := srv.tlsServer.Start()
	return &srv, err
}
