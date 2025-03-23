#!/usr/bin/env node
import {env, exit} from "node:process";
import path from "node:path";
import process from "node:process";
import {ZwaveJSBinding} from "./ZWaveJSBinding.ts";
import BindingConfig from "./BindingConfig.ts";
import ConnectToHub from "../hivelib/messaging/ConnectToHub.ts";
import getLogger from "./getLogger.ts";

const log = getLogger()

process.on("uncaughtException", (err) => {
    log.error("uncaught exception:", err)
})

async function main() {
console.log("Starting hiveot zwavejs binding...")
    //--- Step 1: load config

    // the client ID is application binary. This allows using multiple instances for
    // each client.
    // The launcher automatically creates a token file. To manually create one:
    // > hubcli addsvc zwavejs-1
    const clientID = path.basename(process.argv0)

    const appConfig = new BindingConfig(clientID)

    // MOVE THE FOLLOWING LINE TO CONFIG AFTER INITIAL DEVELOPMENT
    // My Z-stick doesn't handle soft reset
    appConfig.zwDisableSoftReset = true

    // When running from a pkg'ed binary, zwavejs must have a writable copy for device config.
    // Use the storage folder set in app config.
    log.info("storage dir", "path", appConfig.storesDir)
    if (appConfig.storesDir) {
        const hasEnv = env.ZWAVEJS_EXTERNAL_CONFIG
        if (!hasEnv || hasEnv == "") {
            env.ZWAVEJS_EXTERNAL_CONFIG = path.join(appConfig.storesDir, "config")
        }
    }

    //--- Step 2: Connect to the Hub
    let binding: ZwaveJSBinding
    try {
        const hc = await ConnectToHub(appConfig.hubURL, appConfig.loginID, appConfig.caCertPEM, false)
        await hc.connectWithToken(appConfig.loginToken)

        //--- Step 3: Start the binding and zwavejs driver
        binding = new ZwaveJSBinding(hc, appConfig);
        await binding.start();

    } catch(e){
        console.log("Unable to connect to the hub:",e)
        exit(1)
    }

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