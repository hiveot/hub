
import { Bonjour } from 'bonjour-service';
import * as tslog from 'tslog';

const HIVEOT_HUB_SERVICE = "hiveot";
const log = new tslog.Logger()

// locateHub uses DNS-SD to search the hiveot record of the hub gateway for up to 5 seconds.
// If found, it returns with its websocket address wss://<addr>:<port>/<path>
export async function locateHub(): Promise<{ hubURL: string, core: string }> {
    return new Promise((resolve, reject) => {
        const locator = new Bonjour();
        locator.findOne({ type: HIVEOT_HUB_SERVICE }, 5000, function (service: any) {
            if (!service || !service.addresses || service.addresses.length == 0 || !service.txt) {
                reject("service not found");
                return;
            }
            // from nodejs, only websockets can be used for the capnp connection
            let addr = service.addresses[0];
            let kv = service.txt;
            let core = kv["core"];
            let wssPort = kv["wss"];
            let wssPath = kv["path"];
            if (wssPort) {
                addr = "wss://" + addr + ":" + wssPort + wssPath;
            } else {
                addr = kv["rawurl"]
            }
            log.info("found service: ", addr);
            resolve({ hubURL: addr, core: core });
        });
    });
}
