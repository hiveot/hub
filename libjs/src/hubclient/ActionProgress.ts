
// ProgressStatus holds the progress of action request delivery
import {ThingMessage} from "@hivelib/things/ThingMessage";
import {ProgressStatusDelivered, ProgressStatusFailed, ProgressStatusCompleted} from "@hivelib/api/vocab/vocab";

export class ActionProgress extends Object{
    // Thing ID
    thingID: string = ""
    // action name
    name:string = ""
    // Request ID
    messageID: string = ""
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
        this.messageID = msg.messageID
        this.reply = undefined
        this.progress = ProgressStatusDelivered
    }
    // action processing completed, possibly with error
    completed(msg: ThingMessage, reply?:any, err?: Error) {
        this.thingID = msg.thingID
        this.name = msg.name
        this.messageID = msg.messageID
        this.reply = reply
        this.progress = ProgressStatusCompleted
        if (err) {
            this.error = err.name + ": " + err.message
        }
    }
    // unable to deliver to Thing
    failed(msg: ThingMessage, err: Error|string|undefined) {
        this.thingID = msg.thingID
        this.name = msg.name
        this.messageID = msg.messageID
        this.progress = ProgressStatusFailed
        if (err instanceof Error) {
            this.error = err.name + ": " + err.message
        } else {
            this.error = err
        }
    }
}
