
// ProgressStatus holds the progress of action request delivery
import {ThingMessage} from "@hivelib/things/ThingMessage";
import {RequestDelivered, RequestFailed, RequestCompleted} from "@hivelib/api/vocab/vocab";

export class ActionStatus extends Object{
    // Thing ID
    thingID: string = ""
    // action name
    name:string = ""
    // Request ID
    requestID: string = ""
    // Updated delivery status
    progress: string|undefined
    // Error in case delivery status has ended without completion.
    error?: string =""
    // Reply in case delivery status is completed
    reply?: any = undefined

    // action was delivered to agent
    delivered(msg: ThingMessage) {
        this.thingID = msg.thingID
        this.name = msg.name
        this.requestID = msg.requestID
        this.reply = undefined
        this.progress = RequestDelivered
    }
    // action processing completed, possibly with error
    completed(msg: ThingMessage, reply?:any, err?: Error) {
        this.thingID = msg.thingID
        this.name = msg.name
        this.requestID = msg.requestID
        this.reply = reply
        this.progress = RequestCompleted
        if (err) {
            this.error = err.name + ": " + err.message
        }
    }
    // unable to deliver to Thing
    failed(msg: ThingMessage, err: Error|string|undefined) {
        this.thingID = msg.thingID
        this.name = msg.name
        this.requestID = msg.requestID
        this.progress = RequestFailed
        if (err instanceof Error) {
            this.error = err.name + ": " + err.message
        } else {
            this.error = err
        }
    }
}
