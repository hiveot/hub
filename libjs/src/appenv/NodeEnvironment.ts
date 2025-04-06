import fs, { existsSync } from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import {Buffer} from "node:buffer";

import { Logger } from 'tslog'
import yaml from 'yaml';
import { program } from 'commander';

const slog = new Logger({ name: "zwavejs" })

const DEFAULT_CA_CERT_FILE = "caCert.pem"


// AppEnvironment holds the running environment naming conventions.
// Intended for services and plugins.
// This contains folder locations, CA certificate and application clientID
export default class NodeEnvironment extends Object {
    // Directories and files when running a nodejs application
    // Application binary folder, eg launcher, cli, ...
    binDir: string = ""
    // Plugin folder
    pluginsDir: string = ""
    // Home folder, default this is the parent of bin, config, certs and logs
    homeDir: string = ""
    // config folder with application and configuration files
    configDir: string = ""
    // Ca certificates and login key and token location
    certsDir: string = ""
    // client's key pair file location
    keyFile: string = ""
    // client's auth token file location
    tokenFile: string = ""
    // Logging output
    logsDir: string = ""
    // logging level
    loglevel: string = ""
    // Root of the service stores
    storesDir: string = ""

    // schema://address:port/ of the hub. Default is autodetect using DNS-SD
    hubURL: string = ""

    // Credentials
    // CA cert, if loaded, in PEM format, used to verify hub server TLS connection
    caCertPEM: string = ""
    // the clientID. Default is the application binary name - hostname
    // this is the default for the login ID
    clientID: string = ""
    // the client's private/public key pair used in authentication
    clientKey: string = ""
    // the client's login ID, when different from the default clientID
    loginID: string = ""
    // the client's authentication token
    loginToken: string = ""


    // new NodeEnvironment returns the application environment including folders 
    // for use by the Hub services running nodejs.
    //
    // Optionally parse commandline flags:
    //
    //	-home  		alternative home directory. Default is the parent folder of the app binary
    //	-clientID  	alternative clientID. Default is the application binary name.
    //	-config     alternative config directory. Default is home/certs
    //	-configFile alternative application config file. Default is {clientID}.yaml
    //	-loglevel   debug, info, warning (default), error
    //	-server     optional server URL or "" for auto-detect
    //	-core       optional server core or "" for auto-detect
    //
    // The default 'user based' structure is:
    //
    //		home
    //		  |- bin                Core binaries
    //	      |- plugins            Plugin binaries
    //		  |- config             Service configuration yaml files
    //		  |- certs              CA and service certificates
    //		  |- logs               Logging output
    //		  |- run                PID files and sockets
    //		  |- stores
    //		      |- {service}      Store for service
    //
    // The system based folder structure is used when launched from a path starting
    // with /usr or /opt:
    //
    //	/opt/hiveot/bin            Application binaries, cli and launcher
    //	/opt/hiveot/plugins        Plugin binaries
    //	/etc/hiveot/conf.d         Service configuration yaml files
    //	/etc/hiveot/certs          CA and service certificates
    //	/var/log/hiveot            Logging output
    //	/run/hiveot                PID files and sockets
    //	/var/lib/hiveot/{service}  Storage of service
    //
    // This uses os.Args[0] application path to determine the home directory, which is the
    // parent of the application binary.
    // The default clientID is based on the binary name using os.Args[0]. 
    // The actual clientID is used for the configfile {clientID}.yaml, auth key and token file, and storage location.
    //
    //  appID is the application binary name used as the default clientID
    //	homeDir to override the auto-detected or commandline paths. Use "" for defaults.
    //	withFlags parse the commandline flags for -home and -clientID

    public initialize(appID: string, homeDir: string, withFlags: boolean) {
        this.clientID = appID
        this.homeDir = homeDir

        //serverCore := ""

        // startup defaults
        // debugger
        // Try to be smart about whether to use the system structure.
        // If the path starts with /opt or /usr then use
        // the system folder configuration. This might be changed in future if it turns
        // out not to be so smart at all.
        // Future: make this work on windows
        const useSystem = homeDir.startsWith("/usr") || homeDir.startsWith("/opt")
        if (useSystem) {
            this.homeDir = path.join("/var", "lib", "hiveot")
            this.binDir = path.join("/opt", "hiveot")
            this.pluginsDir = path.join(this.binDir, "plugins")
            this.configDir = path.join("/etc", "hiveot", "conf.d")
            this.certsDir = path.join("/etc", "hiveot", "certs")
            this.logsDir = path.join("/var", "log", "hiveot")
            this.storesDir = path.join("/var", "lib", "hiveot")
        } else { // use application parent dir
            //slog.Infof("homeDir is '%s", homeDir)
            this.homeDir = homeDir
            this.binDir = "bin"
            this.certsDir = "certs"
            this.configDir = "config"
            this.logsDir = "logs"
            this.pluginsDir = "plugins"
            this.storesDir = "stores"
        }
        this.loglevel = process.env["LOGLEVEL"] || "warning"
        this.hubURL = ""
        this.clientKey = ""
        this.loginID = ""
        this.loginToken = ""

        // default home folder is the parent of the core or plugin binary
        if (this.homeDir == "") {
            this.binDir = path.dirname(process.argv0)
            if (!path.isAbsolute(this.binDir)) {
                const cwd = process.cwd()
                this.binDir = path.join(cwd, this.binDir)
            }
            this.homeDir = path.join(this.binDir, "..")
        }

        // apply commandline options
        if (withFlags) {
            program
                .name('zwavejs')
                .description("HiveOT binding for the zwave protocol using zwavejs")
                .option('-c, --config <string>', "override the location of the config file ")
                .option('--clientID <string>', "application client ID to authenticate as")
                .option('--home <string>', "override the HiveOT application home directory")
                .option('--certs <string>', "override service auth certificate directory")
                .option('--logs <string>', "override log-files directory")
                .option('--loglevel <string>', "'error', 'warn', 'info', 'debug'")
                .option('--serverURL <string>', "server URL or empty for automatic discovery")
            program.parse();
            const options = program.opts()

            // option '--home' changes all paths
            if (options.home) {
                this.homeDir = options.home
            }
            // option '--clientID' replaces the binary name for use in config and storage folders
            if (options.clientID) {
                this.clientID = options.clientID
            }
            // apply commandline overrides
            if (options.config) {
                // if a configfile is given then load it now as commandline overrides configfile
                this.loadConfigFile(options.config)
            } else {
                const configFile = path.join(this.homeDir, this.configDir, this.clientID + ".yaml")

                // try loading the config file
                this.loadConfigFile(configFile)
            }

            this.certsDir = (options.certs) ? options.certs : this.certsDir
            this.hubURL = options.server ? options.server : this.hubURL
            this.logsDir = (options.logs) ? options.logs : this.logsDir
            this.loglevel = (options.loglevel) ? options.loglevel : "warning"
        } else {
            const configFile = path.join(this.configDir, this.clientID + ".yaml")
            // try loading the config file
            this.loadConfigFile(configFile)
        }

        // make paths absolute
        if (!path.isAbsolute(this.homeDir)) {
            const cwd = process.cwd()
            this.homeDir = path.join(cwd, this.homeDir)
        }
        if (!path.isAbsolute(this.certsDir)) {
            this.certsDir = path.join(this.homeDir, this.certsDir)
        }
        if (!path.isAbsolute(this.configDir)) {
            this.configDir = path.join(this.homeDir, this.configDir)
        }
        // storage has subdir for each service
        if (!path.isAbsolute(this.storesDir)) {
            this.storesDir = path.join(this.homeDir, this.storesDir)
        }
        if (!existsSync(this.storesDir)) {
            // writable for current process only
            fs.mkdirSync(this.storesDir, { mode: 0o700 })
        }


        // load the CA cert if found
        const caCertFile = path.join(this.certsDir, DEFAULT_CA_CERT_FILE)
        try {
            this.caCertPEM = fs.readFileSync(caCertFile).toString()
        } catch {
            this.caCertPEM = ""
        }

        // determine the expected location of the service auth key and token
        this.tokenFile = path.join(this.certsDir, this.clientID + ".token")
        this.keyFile = path.join(this.certsDir, this.clientID + ".key")

        // attempt to load the CA cert, client key, and auth token, if available in a file
        if (!this.caCertPEM) {
            try {
                const caCertFile = path.join(this.certsDir, "caCert.pem")
                this.caCertPEM = fs.readFileSync(caCertFile).toString()
            } catch { }
        }
        // load the client's private key if it exists
        if (!this.clientKey) {
            try {
                this.clientKey = fs.readFileSync(this.keyFile).toString()
                this.clientKey.trim()
            } catch { }
        }
        // allow the config to override the login ID and token
        if (!this.loginID) {
            this.loginID = this.clientID
        }
        if (!this.loginToken) {
            try {
                this.loginToken = fs.readFileSync(this.tokenFile).toString()
                this.loginToken = this.loginToken.trim()
            } catch { }
        }

    }


    // LoadConfigFile loads the application configuration 
    //
    // This throws an error if loading or parsing the config file fails.
    // Returns normally if the config file doesn't exist or is loaded successfully.
    public loadConfigFile(path: string): void {
        let cfgData: Buffer | undefined
        try {
            cfgData = fs.readFileSync(path)
            slog.info("Loaded configuration file", "configFile", path)
        } catch (e) {
            slog.info("Configuration file not found. Ignored.", "path", path)
        }

        if (cfgData) {
            const cfg: object = yaml.parse(cfgData.toString())
            let target: Object = this
            // iterate the attributes and apply
            Object.assign(this, cfg)
        }
    }
}
