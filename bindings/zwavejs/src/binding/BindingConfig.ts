import type {IZWaveConfig} from "./zwapi.js";
import fs from "fs";
import yaml from "yaml";
import crypto from "crypto";
import path from "path";
import {program} from "commander";
import os from "os";


export interface IBindingConfig extends IZWaveConfig {
    logsDir: string
    storesDir: string

    // binding publisher ID. Default is 'zwavejs-{homeID}'
    publisherID: string

    // address and port of the gateway. Default is autodetect using DNS-SD
    gateway: string
    // auth certificates directory
    certsDir: string

    // logging of discovered value IDs to CSV. Intended for testing
    vidCsvFile: string | undefined

    // maximum number of scenes. Default is 10.
    // this reduces it from 255 scenes, which produces massive TD documents
    // For the case where more than 10 is needed, set this to whatever is needed.
    maxNrScenes: number
}

// ParseCommandline options and load config
// This returns the new binding configuration.
export function parseCommandlineConfig(bindingName: string): IBindingConfig {
    let newConfig: IBindingConfig
    // the application binary lives in {home}/bin/services
    let homeDir = path.dirname(path.dirname(path.dirname(process.argv[0])))
    let hostName = os.hostname()

    program
        .name('zwavejs')
        .description("HiveOT binding for the zwave protocol using zwavejs")
        .option('--gateway <string>', "override the websocket address and port of the HiveOT gateway")
        .option('-c --config <string>', "override the location of the config file ")
        .option('--home <string>', "override the HiveOT application home directory")
        .option('--certs <string>', "override service auth certificate directory")
        .option('--logs <string>', "override log-files directory")
        .option('--cache <string>', "override cache directory")
    program.parse();
    const options = program.opts()

    // option '--home' changes all defaults
    if (options.home) {
        homeDir = options.home
    }
    // apply commandline overrides
    let configPath = (options.config) ? options.config : path.join(homeDir, "config", bindingName + ".yaml")
    let gateway = options.gateway ? options.gateway : "" // default is auto discover
    let certsDir = (options.certs) ? options.certs : path.join(homeDir, "certs")
    let logsDir = (options.logs) ? options.logs : path.join(homeDir, "logs")
    let cacheDir = (options.cache) ? options.cache : path.join(homeDir, "stores")

    // load config. Save if it doesn't exist
    // option '--config' changes the config file
    if (!fs.existsSync(configPath)) {
        saveDefaultConfig(configPath, bindingName, gateway, certsDir, logsDir, cacheDir)
    }
    let cfgData = fs.readFileSync(configPath)
    newConfig = yaml.parse(cfgData.toString())

    // apply commandline overrides and ensure proper defaults
    if (options.certs || !newConfig.certsDir) {
        newConfig.certsDir = certsDir
    }
    if (options.logs || !newConfig.logsDir) {
        newConfig.logsDir = logsDir
    }
    if (options.cache || !newConfig.cacheDir) {
        newConfig.cacheDir = cacheDir
    }
    // ensure all required properties exist
    if (!newConfig.publisherID) {
        newConfig.publisherID = bindingName + "-" + hostName
    }
    if (options.gateway) {
        newConfig.gateway = options.gateway // auto discovery
    }
    if (!newConfig.maxNrScenes) {
        newConfig.maxNrScenes = 10
    }
    console.log("home dir=", homeDir)
    return newConfig
}

export function loadCerts(bindingName: string, certsDir: string): [clientCertPem: string, clientKeyPem: string, caCertPem: string] {

    let clientCertFile = path.join(certsDir, bindingName + "Cert.pem")
    let clientKeyFile = path.join(certsDir, bindingName + "Key.pem")
    let caCertFile = path.join(certsDir, "caCert.pem")

    let clientCertPem = fs.readFileSync(clientCertFile)
    let clientKeyPem = fs.readFileSync(clientKeyFile)
    let caCertPem = fs.readFileSync(caCertFile)

    return [clientCertPem.toString(), clientKeyPem.toString(), caCertPem.toString()]
}


// Generate and a default configuration yaml file for the binding.
export function saveDefaultConfig(
    configPath: string, bindingName: string, gateway: string, certsDir: string, logsDir: string, cacheDir: string) {

    let bindingID = bindingName + "-" + os.hostname()
    let ConfigText = "# HiveOT " + bindingName + " binding configuration file\n" +
        "# Generated: " + new Date().toString() + "\n" +
        "\n" +
        "# Binding ID used for publications. \n" +
        "# Multiple instances must use different IDs. Default is binding-hostname\n" +
        "publisherID: " + bindingID + "\n" +
        "\n" +
        "# Gateway connection protocol, address, port. Default is automatic\n" +
        "#gateway: wss://127.0.0.1:9884/ws\n" +
        "\n" +
        "# Optionally write discovered value ID's to a csv file. Intended for troubleshooting.\n" +
        "#vidCsvFile: " + path.join(logsDir, "zwinfo.csv") + "\n" +
        "\n" +
        "# ZWave S2 security keys. 16 Byte hex strings\n" +
        "# Keep these secure to protect your network:\n" +
        "S2_Unauthenticated: " + crypto.randomBytes(16).toString("hex") + "\n" +
        "S2_Authenticated: " + crypto.randomBytes(16).toString("hex") + "\n" +
        "S2_AccessControl: " + crypto.randomBytes(16).toString("hex") + "\n" +
        "S2_Legacy: " + crypto.randomBytes(16).toString("hex") + "\n" +
        "\n" +
        "# Serial port of ZWave USB controller. Default is automatic.\n" +
        "#zwPort: /dev/ttyACM0\n" +
        "\n" +
        "# Optional logging of zwavejs driver\n" +
        "# error, warn, http, info, verbose, or debug\n" +
        "#zwLogFile: " + path.join(logsDir, "zwavejs-driver.log") + "\n" +
        "zwLogLevel: warn\n" +
        "\n" +
        "# Location where the ZWavejs driver stores its data\n" +
        "cacheDir: " + path.join(cacheDir, bindingName) + "\n" +
        "\n" +
        "# Limit max nr of scenes to reduce TD document size. Default is 10\n" +
        "#maxNrScenes: 10\n"
    fs.writeFileSync(configPath, ConfigText, {mode: 0o644})
}
