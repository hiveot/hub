// Definition of Thing TD fields 
// This javascript definition must be kept in sync with the golang hub/lib/client/pkg/vocab definitions

// FIXME: use vocab generated from capnp 

// Standardize @action types for operators.
// By default these are restricted to users with the operator role permissions.
export enum ActionTypes {
    AVChannel = "avChannel",          // <number> preset channel selection
    AVMute = "avMute",                // <boolean> AV mute (true) or unmute (false)
    AVPlay = "avPlay",                // <boolean> AV play (true) or pause (false)
    AVNext = "avNext",                // [] next channel/track
    AVPrev = "avPrev",                // [] previous channel/track
    AVVolume = "avVolume",            // <number> set new AV volume 0.100%
    Lock = "lock",                    // <boolean> lock (true) or unlock (false)
    Switch = "switch",                  // <boolean> switch on (1), off (0)
    Ping = "ping",                    // [] contact the device to see if it is reachable
    Refresh = "refresh",              // [] refresh device info
    Set = "set",                      // <number> set a value
}


// HiveOT device types
export enum DeviceTypes {
    TypeAdapter = "adapter",       // software adapter or service, eg virtual device
    AVControl = "avControl",    // Audio/Video controller
    AVReceiver = "avReceiver",    // Node is a (not so) smart radio/receiver/amp (eg, denon)
    Beacon = "beacon",         // device is a location beacon
    Button = "button",        // device is a physical button device with one or more buttons
    Camera = "camera",       // Node with camera
    CarbonMonoxideDetector = "coDetector",
    Computer = "computer",       // General purpose computer
    Dimmer = "dimmer",      // light dimmer
    DoorWindowSensor = "doorWindowSensor",
    Gateway = "gateway",       // device is a gateway for other nodes (onewire, zwave, etc)
    Keypad = "keypad",     // Entry key pad
    Lock = "lock",    // Electronic door lock
    Multisensor = "multisensor",   // Node with multiple sensors
    NetRepeater = "netRepeater",  // Node is a zwave or other network repeater
    NetRouter = "netRouter",      // Node is a network router
    NetSwitch = "netSwitch",     // Node is a network switch
    NetWifiAP = "wifiAP",    // Node is a wifi access point
    OnOffSwitch = "onOffSwitch",    // Node is a physical on/off switch
    Phone = "phone",      // device is a phone
    PowerMeter = "powerMeter",     // Node is a power meter
    Pushbutton = "pushbutton",    // Node is a push button switch
    SceneController = "sceneController", // Node is a scene controller for other devices
    Sensor = "sensor",     // Node is a single sensor (volt,...)
    Service = "service",     // Node provides a service
    Smartlight = "smartlight",    // Node is a smart light, eg philips hue
    SmokeDetector = "smokeDetector",  // Node is a smoke detector
    Thermometer = "thermometer", // Node is a temperature meter
    Thermostat = "thermostat",   // Node is a thermostat control unit
    TV = "tv",      // Node is a (not so) smart TV
    Unknown = "unknown",       // type not identified
    WaterValve = "waterValve",     // Water valve control unit
    WeatherService = "weatherService",// Node is a service providing current and forecasted weather
    WeatherStation = "weatherStation",// Node is a weatherstation device
    WeighScale = "weighScale",   // Node is an electronic weight scale
}


// Standardized sensor event @type values.
// Intended for grouping events of similar types.
export enum EventTypes {
    //
    Properties = "properties", // standardized event with JSON properties key-value map
    TD = "td", // standardized event with JSON TD document
    //
    Acceleration = "acceleration",  // <boolean> acceleration in m/s2
    AirQuality = "airQuality",      // <number> in 1-10?
    Alarm = "alarm",                // <boolean> other alarm state changed
    AtmosphericPressure = "atmosphericPressure", // <number> in mbar (or psi)
    CarbonDioxideLevel = "co2level", // <number>
    CarbonMonoxideLevel = "coLevel", // <number>
    Current = "current",         // <number> Electrical current report in A
    Dewpoint = "dewpoint",       // <number> dew point in degrees
    Energy = "energy",           // <number> Energy report in KWH
    HeatIndex = "heatIndex",     // <number> heat index in degrees
    Humidex = "humidity",        // <number> humidex in degrees
    LatLon = "latlon",           // [lat, lon] location change
    LowBattery = "lowBattery",   // <boolean> low battery alarm enabled/disabled
    Luminance = "luminance",     // <number> luminance sensor in ?
    Motion = "motion",           // [boolean] motion on/off detected
    Power = "power",             // <number> power report in Watt
    PushButton = "pushButton",   // [<number>] pushbutton <N> was pressed. Where N is for multi-switches
    Switch = "switch",           // <boolean> binary switch status changed
    Scene = "scene",             // <number> scene activated by scene controller
    Status = "status",           // <string> node status "awake", "sleeping", "dead", "alive"
    Sound = "sound",             // <number> sound detected with loudness
    Temperature = "temperature", // <number> temperature report in degrees C (or F or K)
    UV = "ultraviolet",          // <number> UV sensor in ?
    Vibration = "vibration",     // <boolean> vibration sensor alarm started (true), ended (false)
    Value = "value",             // <number> generic sensor value event
    Voltage = "voltage",         // <number> electrical voltage report in V
    WaterLevel = "waterLevel",   // <number> Water level report in meters (or yards, feet, ...)
    WindHeading = "windHeading", // <number> Wind direction in 0-359 degrees
    WindSpeed = "windSpeed",     // <number> Wind speed in m/s (or kph, mph)
}

// standardized property attribute @types from HiveOT vocabulary
export enum PropTypes {
    AlarmType = "alarmType",             // <string> from enum
    CPULevel = "cpuLevel",               // <number> in %
    DateTime = "dateTime",               // <string> ISO8601
    BatteryLevel = "batteryLevel",       // <number> in %
    FirmwareVersion = "firmwareVersion", // <string>
    HardwareVersion = "hardwareVersion", // <string> version of the physical device
    IPAddress = "ipAddress",
    Latency = "latency",                 // <number> in msec (or usec, seconds)
    MACAddress = "macAddress",
    Manufacturer = "manufacturer",       // <string> device manufacturer
    Name = "name",                       // device name
    ProductName = "productName",         // <string>
    SignalStrength = "signalStrength",   // <number> in dBm (dB millivolts per meter)
    SoftwareVersion = "softwareVersion", // <string> application software
}

/**
 * Standardized property/event names to be used by Things and plugins to build their TD
 * If a device has multiple instances of a property (multi button, multi temperature) with
 * the same name then it becomes an object with nested properties for each instance.
 */
// export const VocabAddress = "address" // device domain or ip address
// export const VocabBatch = "batch" // Batch publishing size
// export const VocabChannel = "avChannel"
// export const VocabColor = "color" // Color in hex notation
// export const VocabColorTemperature = "colortemperature"
// export const VocabConnections = "connections"
// export const VocabDisabled = "disabled" // device or sensor is disabled
// export const VocabDuration = "duration"
// export const VocabErrors = "errors"
// //
// export const VocabFilename = "filename"       // [string] filename to write images or other values to
// export const VocabGatewayAddress = "gatewayAddress" // [string] the 3rd party gateway address
// export const VocabHostname = "hostname"       // [string] network device hostname
// export const VocabHue = "hue"            //
// // export const VocabHumidex = "humidex"        // <number> unit=C or F
// // export const VocabHumidity = "humidity"       // [string] %
// export const VocabImage = "image"            // [byteArray] unit=jpg,gif,png
// export const VocabLocationName = "locationName"   // [string] name of a location
// export const VocabLoginName = "loginName"      // [string] login name to connect to the device. Value is not published
// export const VocabMAC = "mac"            // [string] MAC address for IP nodes
// export const VocabName = "name"           // [string] Name of device or service
// export const VocabNetmask = "netmask"        // [string] IP network mask
// export const VocabPassword = "password" // password to connect. Value is not published.
// export const VocabPollInterval = "pollInterval" // polling interval in seconds
// export const VocabPort = "port"         // network address port
// export const VocabPowerSource = "powerSource"  // battery, usb, mains
// export const VocabPublicKey = "publicKey"    // public key for encrypting sensitive configuration settings
// // export const VocabRain = "rain"
// export const VocabRelay = "relay"
// export const VocabSaturation = "saturation"
// export const VocabScale = "scale"
// export const VocabSnow = "snow"
// export const VocabSubnet = "subnet" // IP subnets configuration
// export const VocabUnknown = ""    // Not a known output
// export const VocabURL = "url" // node URL


/**
 * The following terms are defined in the WoT Thing Description definition
 */
// WoT data schema
export const TDAtType = "@type"
export const TDAtContext = "@context"
export const TDAnyURI = "https://www.w3.org/2019/wot/td/v1"
export const TDActions = "actions"
export const TDCreated = "created"
export const TDDescription = "description"
export const TDDescriptions = "descriptions"
export const TDEvents = "events"
export const TDForms = "forms"
export const TDID = "id"
export const TDLinks = "links"
export const TDProperties = "properties"
export const TDSecurity = "security"
export const TDSupport = "support"
export const TDTitle = "title"
export const TDTitles = "titles"
export const TDVersion = "version"

// additional data schema vocab
export const TDConst = "const"

export enum DataType {
    Bool = "boolean",
    AnyURI = "anyURI",
    Array = "array",
    DateTime = "dateTime",
    Integer = "integer",
    Number = "number",
    Object = "object",
    String = "string",
    UnsignedInt = "unsignedInt",
    Unknown = ""
}

// WoTDouble              = "double" // min, max of number are doubles
export const TDEnum = "enum"
export const TDFormat = "format"
export const TDHref = "href"
export const TDInput = "input"
export const TDMaximum = "maximum"
export const TDMaxItems = "maxItems"
export const TDMaxLength = "maxLength"
export const TDMinimum = "minimum"
export const TDMinItems = "minItems"
export const TDMinLength = "minLength"
export const TDModified = "modified"
export const TDOperation = "op"
export const TDOutput = "output"
export const TDReadOnly = "readOnly"
export const TDRequired = "required"
export const TDWriteOnly = "writeOnly"
export const TDUnit = "unit"
