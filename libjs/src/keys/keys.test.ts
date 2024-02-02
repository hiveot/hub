
// test1: generate, export and import an ECDSA key pair

import { ECDSAKey } from "./ECDSAKey";
// import {Ed25519Key} from "./Ed25519Key.js";
import { natsKey } from "./natsKey";
import { IHiveKey } from './IHiveKey';


function newKey(): IHiveKey {
    return new ECDSAKey()
    // return new nkeysKey()
}


async function test1() {
    const message = "hello world"

    let keys1 = newKey()
    // let keys1 = new EcdsaKeys()
    keys1.initialize()

    let privPEM = keys1.exportPrivate()
    let pubPEM = keys1.exportPublic()

    let keys2 = newKey()
    keys2.importPrivate(privPEM)
    keys2.importPublic(pubPEM)

    let msgBuf = Buffer.from(message)
    let signature = keys1.sign(msgBuf)
    let verified = keys2.verify(signature, msgBuf)
    if (!verified) {
        throw ("test failed")
    }
    console.log("test successful")
}


test1()