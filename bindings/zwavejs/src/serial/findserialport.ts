
// Determine which serial port is available
import fs from "node:fs";
import path from "node:path";

// return the first serial port using /dev/serial/by-id.
export default function findSerialPort(): string {
    const serialDir = "/dev/serial/by-id/"
    try {
        const serdir = fs.opendirSync(serialDir);
        try {
            const first = serdir.readSync()
            if (first != null) {
                return path.join(serialDir, first.name)
            }
        }
        finally {
            serdir.close()
        }
    } catch (err) {
        console.error(err);
    }

    // force an error
    return "/dev/serialportnotfound"
}
