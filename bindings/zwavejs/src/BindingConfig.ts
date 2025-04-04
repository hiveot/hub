import fs, { existsSync } from "node:fs";
import path from "node:path";
import type { IZWaveConfig } from "./ZWAPI.ts";
import  NodeEnvironment from "../hivelib/appenv/NodeEnvironment.ts";


// This binding's service configuration  
export default class BindingConfig extends NodeEnvironment implements IZWaveConfig {
    // zwave network keys
    S0_Legacy: string | undefined
    S2_AccessControl: string | undefined
    S2_Authenticated: string | undefined
    S2_Unauthenticated: string | undefined
    S2LR_AccessControl: string | undefined
    S2LR_Authenticated: string | undefined
    //
    zwDisableSoftReset: boolean = true  // disable the soft reset if driver has trouble connecting
    zwPort: string | undefined          // controller port: ""=auto, /dev/ttyACM0, ...
    zwLogFile: string | undefined       // driver logfile if any
    // driver log level, "" no logging
    zwLogLevel: "error" | "warn" | "info" | "verbose" | "debug" | "" = "warn"
    // cacheDir where zwavejs stores its discovered node info
    cacheDir: string | undefined        // alternate storage directory
    // flag, only publish changed values on value update
    publishOnlyChanges: boolean = false

    // logging of discovered value IDs to CSV. Intended for testing
    vidCsvFile: string | undefined

    // maximum number of scenes. Default is 10.
    // this reduces it from 255 scenes, which produces massive TD documents.
    // For the case where more than 10 is needed, set this to whatever is needed.
    maxNrScenes: number = 10

    constructor(clientID: string) {
        super()
        const homeDir = ""
        const withFlags = true

        this.initialize(clientID, homeDir, withFlags)
        // zwave storage cache directory uses the storage directory
        this.cacheDir = path.join(this.storesDir, this.clientID)
        if (!existsSync(this.cacheDir)) {
            // writable for current process only
            fs.mkdirSync(this.cacheDir, { mode: 0o700 })
        }

        // last, the zwave driver needs security keys
    }
}



