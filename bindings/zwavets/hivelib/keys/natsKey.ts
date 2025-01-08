
// nats nkey based implementation using nodeJS
import type { IHiveKey } from "./IHiveKey";

import * as nkeys from "nkeys.js"

export class natsKey implements IHiveKey {
    kp: nkeys.KeyPair
    pubKey: string

    constructor() {
        this.kp = nkeys.createUser()
        this.pubKey = this.kp.getPublicKey()
    }

    // exportPrivate returns the encoded private key if available
    public exportPrivate(): string {

        if (!this.kp) {
            throw ("private key not created or imported")
        }
        let seed = this.kp.getSeed()
        return seed.toString()
    }

    // exportPublic returns the encoded public key if available
    public exportPublic(): string {
        if (!this.pubKey) {
            throw ("public key not created or imported")
        }
        return this.pubKey
    }

    // importPrivate reads the key-pair from the nkey seed.
    // This throws an error if the encoding is not a valid key
    public importPrivate(seedStr: string): IHiveKey {
        let seedEnc = new TextEncoder().encode(seedStr)
        nkeys.fromSeed(seedEnc)
        return this
    }

    // importPublic reads the public key from the encoded data.
    // This throws an error if the encoding is not a valid public key
    public importPublic(publicStr: string): IHiveKey {
        this.pubKey = publicStr
        this.kp = nkeys.fromPublic(publicStr)
        return this
    }

    // initialize generates a new key set using its curve algorithm
    public initialize(): IHiveKey {
        let kp = nkeys.createUser()
        this.pubKey = kp.getPublicKey()
        return this
    }

    // return the signature of a message signed using this key
    // this requires a private key to be created or imported
    public sign(message: Buffer): Buffer {
        if (!this.kp) {
            throw ("key not created or imported")
        }
        // algorithm depends on key type. sha256 is not used in ed25519
        let sigBuf = this.kp.sign(message)
        return Buffer.from(sigBuf)
    }

    // verify the signature of a message using this key's public key
    // this requires a public key to be created or imported
    // returns true if the signature is valid for the message
    public verify(signature: Buffer, message: Buffer): boolean {
        if (!this.pubKey) {
            throw ("public key not created or imported")
        }
        let isValid = this.kp.verify(message, signature)
        return isValid
    }
}
