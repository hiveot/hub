import {IAgentClient} from "@hivelib/hubclient/IAgentClient";
import {ActionProgress} from "@hivelib/hubclient/ActionProgress";

// duplicated from stateapi.go
const AgentID = "state"      // the state binding agentID
const StateStoreID = "store" // thingID of the storage service
const DeleteMethod = "delete"
const GetMethod = "get"
const GetMultipleMethod = "getMultiple"
const SetMethod = "set"
const SetMultipleMethod = "setMultiple"


// Marshaller for invoke state service method using the given hub client
export class StateClient {
    hc: IAgentClient
    serviceID: string // digitwin ID of the state service

    constructor(hc: IAgentClient) {
        this.hc = hc
        // this.serviceID = makeDigiTwinThingID(AgentID, StateStoreID)
        this.serviceID = "dtw:"+AgentID+":"+ StateStoreID
    }

    // Delete a key
    async Delete(key: string) {
        let args = {
            key: key
        }
        return await this.hc.rpc(this.serviceID, DeleteMethod, args)
    }

    // Get the value of a key
    async Get(key: string): Promise<{ value: string, found: boolean }> {
        let args = {
            key: key
        }
        type RespType = {
            key: string
            found: boolean
            value: string
        }
        let resp:RespType
        try {
            resp = await this.hc.rpc(this.serviceID, GetMethod, args)
        } catch(e) {
            console.error("state.Get error",e)
            throw("state.Get error"+e)
        }
        return { value: resp.value, found: resp.found }
    }

    // Get a map of key:value 
    async GetMultiple(keys: string[]): Promise<{ [index: string]: string }> {
        let args = {
            keys: keys
        }
        type RespType = {
            kv: { [index: string]: string }
        }
        let resp: RespType = await this.hc.rpc(this.serviceID, GetMultipleMethod, args)

        return resp.kv
    }

    // Set the value of a key
    async Set(key: string, data: string):Promise<ActionProgress> {
        let args = {
            key: key,
            value: data
        }
        let stat = await this.hc.rpc(this.serviceID, SetMethod, args)
        return stat
    }

    // Set multiple values at once
    async SetMultiple(kv: { [index: string]: string }) {
        let args = {
            kv: kv
        }
        return await this.hc.rpc(this.serviceID, SetMultipleMethod, args)
    }
}

