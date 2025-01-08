
// ED25519 keys implementation using nodeJS
import type { IHiveKey } from "./IHiveKey";
import crypto from "crypto";

export class Ed25519Key implements IHiveKey {
    privKey: crypto.KeyObject | undefined
    pubKey: crypto.KeyObject | undefined

    constructor() {
    }

    // exportPrivate returns the encoded private key if available
    public exportPrivate(): string {
        if (!this.privKey) {
            throw ("private key not created or imported")
        }
        let privPEM = this.privKey.export({
            format: "pem", // pem, der or jwk
            type: "pkcs8",  // or sec1
        })
        return privPEM.toString()
    }

    // exportPublic returns the encoded public key if available
    public exportPublic(): string {
        if (!this.pubKey) {
            throw ("public key not created or imported")
        }
        let pubPEM = this.pubKey.export({
            format: "pem", // pem, der or jwk
            type: "spki",
        })
        return pubPEM.toString()
    }

    // importPrivate reads the key-pair from the encoded private key
    // This throws an error if the encoding is not a valid key
    public importPrivate(privateEnc: string): IHiveKey {
        // cool! crypto does all the work
        this.privKey = crypto.createPrivateKey(privateEnc)
        this.pubKey = crypto.createPublicKey(privateEnc)
        return this
    }

    // importPublic reads the public key from the encoded data.
    // This throws an error if the encoding is not a valid public key
    public importPublic(publicEnc: string): IHiveKey {
        this.pubKey = crypto.createPublicKey(publicEnc)
        return this
    }


    // initialize generates a new key set using its curve algorithm
    public initialize(): IHiveKey {
        let kp = crypto.generateKeyPairSync("ed25519")
        this.privKey = kp.privateKey
        this.pubKey = kp.publicKey
        return this
    }

    // return the signature of a message signed using this key
    // this requires a private key to be created or imported
    public sign(message: Buffer): Buffer {
        if (!this.privKey) {
            throw ("private key not created or imported")
        }
        // algorithm depends on key type. sha256 is not used in ed25519
        let sigBuf = crypto.sign(null, message, this.privKey)
        return sigBuf
    }

    // verify the signature of a message using this key's public key
    // this requires a public key to be created or imported
    // returns true if the signature is valid for the message
    public verify(signature: Buffer, message: Buffer): boolean {
        if (!this.pubKey) {
            throw ("public key not created or imported")
        }
        let isValid = crypto.verify(null, message, this.pubKey, signature)
        return isValid
    }
}
