package svcconfig

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"path"
	"path/filepath"
)

// SetupFolderConfig creates a folder configuration for based on commandline options.
// Returns the folders, <serviceName>Cert.pem TLS certificate, and caCert.pem CA Certificate, if found.
//
// This invokes flag.Parse(). Flag commandline options added are:
//
//	 -c configFile
//	 --home directory
//	 --certs directory
//	 --services directory
//	 --logs directory
//	 --loglevel info|warning
//	 --run directory
//
//	If a 'cfg' interface is provided, the configuration is loaded from file and parsed as yaml.
//
//	serviceName is used for the configuration file with the '.yaml' extension
func SetupFolderConfig(serviceName string) (f AppFolders, svcCert *tls.Certificate, caCert *x509.Certificate) {

	// run the commandline options
	var certsFolder = ""
	var configFolder = ""
	var homeFolder = ""
	var logLevel = "info"
	var logsFolder = ""
	var runFolder = ""
	var pluginsFolder = ""
	var storesFolder = ""
	var cfgFile = ""

	f = GetFolders(homeFolder, false)
	flag.StringVar(&homeFolder, "home", f.Home, "Application home directory")
	flag.StringVar(&certsFolder, "certs", f.Certs, "Certificates directory")
	flag.StringVar(&configFolder, "config", f.Config, "Configuration directory")
	flag.StringVar(&cfgFile, "c", "", "Service config file")
	flag.StringVar(&pluginsFolder, "plugins", f.Plugins, "Application services directory")
	flag.StringVar(&logsFolder, "logs", f.Logs, "Service log files directory")
	flag.StringVar(&logLevel, "loglevel", logLevel, "Loglevel info|warning. Default is warning")
	flag.StringVar(&runFolder, "run", f.Run, "Runtime directory for sockets and pid files")
	flag.StringVar(&storesFolder, "stores", f.Stores, "Storage directory")
	flag.Parse()

	// homefolder is special as it overrides all default folders
	// detect the override by comparing original folder with assigned folder
	f2 := GetFolders(homeFolder, false)
	if certsFolder != f.Certs {
		f2.Certs = certsFolder
	}
	if configFolder != f.Config {
		f2.Config = configFolder
	}
	if cfgFile != "" {
		f2.ConfigFile = cfgFile
	} else {
		f2.ConfigFile = path.Join(f2.Config, serviceName+".yaml")
	}
	if pluginsFolder != f.Plugins {
		f2.Plugins = pluginsFolder
	}
	if logsFolder != f.Logs {
		f2.Logs = logsFolder
	}
	if runFolder != f.Run {
		f2.Run = runFolder
	}
	f2.SocketPath = filepath.Join(f2.Run, serviceName+".socket")

	if storesFolder != f.Stores {
		f2.Stores = storesFolder
	}
	if logsFolder != "" {
		logFile := path.Join(logsFolder, serviceName+".log")
		logging.SetLogging(logLevel, logFile)
	} else {
		logging.SetLogging(logLevel, "")
	}

	// load the certificates if available
	caCertPath := path.Join(f2.Certs, "caCert.pem")
	caCert, _ = certs.LoadX509CertFromPEM(caCertPath)
	svcCertPath := path.Join(f2.Certs, serviceName+"Cert.pem")
	svcKeyPath := path.Join(f2.Certs, serviceName+"Key.pem")
	svcCert, _ = certs.LoadTLSCertFromPEM(svcCertPath, svcKeyPath)

	return f2, svcCert, caCert
}
