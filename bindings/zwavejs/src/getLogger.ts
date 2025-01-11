// return the default logger

import * as tslog from "tslog";

const defaultLogger = new tslog.Logger({prettyLogTimeZone:"local"})

export function getlogger() { return defaultLogger}