import * as tslog from 'tslog';
import {EventHandler} from "@hivelib/hubclient/IConsumerClient";
import {ThingMessage} from "@hivelib/things/ThingMessage";
// var EventSource = require('eventsource')
import EventSource from 'eventsource'

const hclog = new tslog.Logger()

// import EventSource from 'eventsource'

export function  connectSSE(
    baseURL:string, ssePath:string, authToken:string,handler: EventHandler ):EventSource {

    var eventSourceInitDict = {
        headers: {
            Authorization: 'bearer '+authToken,
            Origin: baseURL,
            "Content-Type": "application/json",
        },
        https:{rejectUnauthorized:false}
    };

    let sseURL = baseURL + ssePath
    const source = new EventSource(sseURL,eventSourceInitDict)

    source.addEventListener("action",(e:any)=>{
       let msg: ThingMessage = JSON.parse(e.data)
        handler(msg)
    })
    source.addEventListener("event",(e:any)=>{
        let msg: ThingMessage = JSON.parse(e.data)
        handler(msg)
    })
    source.addEventListener("property",(e:any)=>{
        let msg: ThingMessage = JSON.parse(e.data)
        handler(msg)
    })
    source.addEventListener("close",(e:any)=>{
        hclog.info("On close", e.data)
    })
    source.addEventListener("error",(e:any)=>{
        hclog.info("On error", e.data)
    })
    source.onmessage = function (msg:any) {
        hclog.info("RECEIVED", msg.data)
    }
    source.onerror = function (err:any){
        hclog.error("Connection error: "+err.message)
    }

    return source
}