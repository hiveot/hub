import {Buffer} from "node:buffer";

// test1: generate, export and import an ECDSA key pair

import { Ed25519Key } from "./Ed25519Key.ts";
// import {Ed25519Key} from "./Ed25519Key.js";
// import { natsKey } from "./natsKey";
import { type IHiveKey } from './IHiveKey.ts';


function newKey(): IHiveKey {
    return new Ed25519Key()
    // return new nkeysKey()
}


async function test1() {
    const message = "hello world"

    const keys1 = newKey()
    // let keys1 = new EcdsaKeys()
    keys1.initialize()

    const privPEM = keys1.exportPrivate()
    const pubPEM = keys1.exportPublic()

    const keys2 = newKey()
    keys2.importPrivate(privPEM)
    keys2.importPublic(pubPEM)

    const msgBuf = Buffer.from(message)
    const signature = keys1.sign(msgBuf)
    const verified = keys2.verify(signature, msgBuf)
    if (!verified) {
        throw ("test failed")
    }
    console.log("test successful")
}


test1()