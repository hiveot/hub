#!/usr/bin/env node
import "./lib/hubapi.js";
import {env, exit} from "process";
import {HubAPI} from "./lib/hubapi.js";
import {loadCerts, parseCommandlineConfig} from "./binding/BindingConfig.js";
import {ZwaveBinding} from "./binding/binding.js";
import path from "path";


async function main() {

//--- Step 1: load config
    const BINDING_NAME = "zwavejs"
// optional override or preset of gateway on first use
    let appConfig = parseCommandlineConfig(BINDING_NAME)
    // FIXME: use bindingID from config
    let [clientCertPem, clientKeyPem, caCertPem] = loadCerts(BINDING_NAME, appConfig.certsDir)
    // When running from a pkg'ed binary, zwavejs must have a writable copy for device config. Use the cache folder
    // if set in app config.
    console.log("cache dir=", appConfig.cacheDir)
    if (appConfig.cacheDir) {
        let hasEnv = env.ZWAVEJS_EXTERNAL_CONFIG
        if (!hasEnv || hasEnv == "") {
            env.ZWAVEJS_EXTERNAL_CONFIG = path.join(appConfig.cacheDir, "config")
        }
    }

//--- Step 2: Connect to the Hub using websockets via wasm/go
    let hapi = new HubAPI()
    await hapi.initialize()
    if (!appConfig.gateway) {
        appConfig.gateway = await hapi.locateHub()
    }
    await hapi.connect(appConfig.publisherID, appConfig.gateway, clientCertPem, clientKeyPem, caCertPem)

//--- Step 3: Start the binding and zwavejs driver
    let binding = new ZwaveBinding(hapi, appConfig);
    await binding.start();

//--- Step 4: Wait for  SIGINT or SIGTERM signal to stop
    console.log("Ready. Waiting for signal to terminate")
    for (const signal of ["SIGINT", "SIGTERM"]) {
        process.on(signal, async () => {
            gostop();
            await binding.stop();
            exit(0);
        });
    }
}


main()