// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// import wasm is preferred as it includes the file in the bundler
// unfortunately it doesn't work running with node or running in the debugger
//import hapiWasm from "./hapi.wasm"

import fs from "fs"
import crypto from "crypto"
import util from "util"
import os from "os"

// polyfills for running in nodejs
globalThis.fs = fs;
globalThis.TextEncoder = util.TextEncoder;
globalThis.TextDecoder = util.TextDecoder;

globalThis.crypto = {
    getRandomValues(b) {
        crypto.randomFillSync(b);
    },
};


export const startWasm = async function (fileName) {

    // import wasm_exec after setting globalThis. Use import function.
    // note that this can fail if the env vars exceed 8192 bytes.
    // workaround: edit wasm_exec.js:508 to exclude XDG_ variables as these are big
    await import("./wasm_exec.js")
    const go = new globalThis.Go();

// go.argv = process.argv.slice(2);
    go.env = Object.assign({TMPDIR: os.tmpdir()}, process.env);
    go.exit = process.exit;
    let wasmdata = fs.readFileSync(fileName)

    return WebAssembly.instantiate(wasmdata, go.importObject)
        .then((result) => {
            process.on("exit", (code) => { // Node.js exits if no event handler is pending
                if (code === 0 && !go.exited) {
                    // deadlock, make Go print error and stack traces
                    go._pendingEvent = {id: 0};
                    go._resume();
                }
            });
            go.run(result.instance);
        }).catch((err) => {
            console.error(err);
            process.exit(1);
        });
}

