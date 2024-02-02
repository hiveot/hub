import type { IZWaveConfig } from "./ZWAPI.js";
import fs, { existsSync } from "fs";
import crypto from "crypto";
import path from "path";
import os from "os";
import { NodeEnvironment } from "@hivelib/appenv/NodeEnvironment.js";
import { homedir } from 'os';


// This binding's service configuration  
export class BindingConfig extends NodeEnvironment implements IZWaveConfig {
    // zwave network keys
    S2_Unauthenticated: string | undefined
    S2_Authenticated: string | undefined
    S2_AccessControl: string | undefined
    S0_Legacy: string | undefined
    //
    zwDisableSoftReset: boolean | undefined  // disable the soft reset if driver fails to connect 
    zwPort: string | undefined               // controller port: ""=auto, /dev/ttyACM0, ...
    zwLogFile: string | undefined            // driver logfile if any
    // driver log level, "" no logging
    zwLogLevel: "error" | "warn" | "info" | "verbose" | "debug" | "" = "warn"
    // cacheDir where zwavejs stores its discovered node info
    cacheDir: string | undefined        // alternate storage directory
    // flag, only publish changed values on value update
    publishOnlyChanges: boolean = true

    // logging of discovered value IDs to CSV. Intended for testing
    vidCsvFile: string | undefined

    // maximum number of scenes. Default is 10.
    // this reduces it from 255 scenes, which produces massive TD documents.
    // For the case where more than 10 is needed, set this to whatever is needed.
    maxNrScenes: number = 10

    constructor(clientID: string) {
        super()
        let homeDir = ""
        let withFlags = true

        this.initialize(clientID, homeDir, withFlags)
        // zwave storage cache directory uses the storage directory
        this.cacheDir = path.join(this.storesDir, this.clientID)
        if (!existsSync(this.cacheDir)) {
            // writable for current process only
            fs.mkdirSync(this.cacheDir, { mode: 0o700 })
        }
    }
}



