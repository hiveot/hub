import process from "node:process";

//--- Step 4: Wait for  SIGINT or SIGTERM signal to stop
async function WaitForSignal() {
    console.log("Ready. Waiting for signal to terminate")
    for (const signal of ["SIGINT", "SIGTERM"]) {
        process.on(signal, async () => {
            // await binding.stop();
            return;
        });
    }
}
