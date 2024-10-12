
// DeliveryStatus holds the progress of action request delivery
import {ThingMessage} from "@hivelib/things/ThingMessage";
import {ProgressStatusDelivered, ProgressStatusFailed, ProgressStatusCompleted} from "@hivelib/api/vocab/vocab";

export class DeliveryStatus extends Object{
    // Request ID
    messageID: string = ""
    // Updated delivery status
    progress: string|undefined
    // Error in case delivery status has ended without completion.
    error?: string =""
    // Reply in case delivery status is completed
    reply?: any =""

    delivered(msg: ThingMessage) {
        this.messageID = msg.messageID
        this.reply = undefined
        this.progress = ProgressStatusDelivered
    }
    completed(msg: ThingMessage, reply?:any, err?: Error) {
        this.messageID = msg.messageID
        this.reply = reply
        this.progress = ProgressStatusCompleted
        if (err) {
            this.error = err.name + ": " + err.message
        }
    }
    failed(msg: ThingMessage, err: Error|string|undefined) {
        this.messageID = msg.messageID
        this.progress = ProgressStatusFailed
        if (err instanceof Error) {
            this.error = err.name + ": " + err.message
        } else {
            this.error = err
        }
    }
}
