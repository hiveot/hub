
import { Bonjour } from 'bonjour-service';
import * as tslog from 'tslog';
import {ProtocolTypeHiveotSSE, ProtocolTypeHiveotWSS} from "@hivelib/transports/IConsumerConnection";

const HIVEOT_HUB_SERVICE = "hiveot";
const log = new  tslog.Logger({prettyLogTimeZone:"local"})

// locateHub uses DNS-SD to search the hiveot record of the hub gateway for up to 5 seconds.
// If found, it returns with the hub connection address:
//    https://<addr>:<port>/<ssepath>
//    wss://<addr>:<port>/<wsspath>
export async function locateHub(): Promise<{
    addr?:string, hiveotSseURL?: string, hiveotWssURL?:string,
    wotWssURL?:string }> {

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
            let wotWssURL = kv[ProtocolTypeHiveotWSS];
            let hiveotWssURL = kv[ProtocolTypeHiveotWSS];
            let hiveotSseURL = kv[ProtocolTypeHiveotSSE];
            // let mqttWssURL = kv[ProtocolTypeHiveotMQTT];
            // let mqttTcpURL = kv["mqtt-tcp"];

            log.info("found service: ", addr);
            resolve({
                addr:addr,
                hiveotSseURL: hiveotSseURL,
                wotWssURL:wotWssURL,
                hiveotWssURL:hiveotWssURL,
                // mqttTcpURL:mqttTcpURL,
            });
        });
    });
}
