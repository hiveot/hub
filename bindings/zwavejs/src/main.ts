#!/usr/bin/env node
import { env, exit } from "process";
import { NewHubClient } from "@hivelib/hubclient/HubClient"
import { ZwaveJSBinding } from "./ZWaveJSBinding";
import path from "path";
import { locateHub } from "@hivelib/hubclient/locateHub";
import fs from "fs";
import { BindingConfig } from "./BindingConfig";
import * as tslog from 'tslog';
const log = new tslog.Logger({ name: "zwavejs" })

process.on("uncaughtException", (err) => {
    log.error("uncaught exception")
})

async function main() {

    //--- Step 1: load config
    // let clientID = "zwavejs"
    // the application name is the clientID
    let clientID = path.basename(process.argv0)

    let appConfig = new BindingConfig(clientID)

    // REMOVE THE FOLLOWING LINE AFTER INITIAL DEVELOPMENT
    // My Z-stick doesn't handle soft reset
    appConfig.zwDisableSoftReset = true


    // When running from a pkg'ed binary, zwavejs must have a writable copy for device config. 
    // Use the storage folder set in app config.
    log.info("storage dir", "path", appConfig.storesDir)
    if (appConfig.storesDir) {
        let hasEnv = env.ZWAVEJS_EXTERNAL_CONFIG
        if (!hasEnv || hasEnv == "") {
            env.ZWAVEJS_EXTERNAL_CONFIG = path.join(appConfig.storesDir, "config")
        }
    }

    //--- Step 2: Connect to the Hub
    let core = ""
    if (!appConfig.hubURL) {
        let uc = await locateHub()
        appConfig.hubURL = uc.hubURL
        core = uc.core
    }
    let hc = NewHubClient(appConfig.hubURL, appConfig.loginID, appConfig.caCertPEM, core)

    // need a key to connect, load or create it
    // note that the HubClient determines the key type
    let kp = hc.createKeyPair()
    if (appConfig.clientKey) {
        kp.importPrivate(appConfig.clientKey)
    } else {
        fs.writeFileSync(appConfig.keyFile, kp.exportPrivate())
    }
    await hc.connectWithToken(kp, appConfig.loginToken)

    //--- Step 3: Start the binding and zwavejs driver
    let binding = new ZwaveJSBinding(hc, appConfig);

    await binding.start();

    //--- Step 4: Wait for  SIGINT or SIGTERM signal to stop
    log.info("Ready. Waiting for signal to terminate")
    for (const signal of ["SIGINT", "SIGTERM"]) {
        process.on(signal, async () => {
            await binding.stop();
            exit(0);
        });
    }
}


main()