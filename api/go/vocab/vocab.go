// Package vocab with HiveOT and WoT vocabulary names for TD Things, properties, events and actions
// DO NOT EDIT. This file is generated and changes will be overwritten
package vocab

// type: WoTVocab
// version: 0.1
// generated: 05 Feb 25 17:42 PST
// source: github.com/hiveot/hub/api/vocab/wot-vocab.yaml
// description: WoT vocabulary definition. See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
const (
  HTOpLogin = "login"
  HTOpLogout = "logout"
  HTOpReadAllEvents = "readallevents"
  HTOpReadAllTDs = "readalltds"
  HTOpReadEvent = "readevent"
  HTOpReadTD = "readtd"
  HTOpRefresh = "refresh"
  HTOpUpdateTD = "updatetd"
  OpCancelAction = "cancelaction"
  OpInvokeAction = "invokeaction"
  OpObserveAllProperties = "observeallproperties"
  OpObserveProperty = "observeproperty"
  OpQueryAction = "queryaction"
  OpQueryAllActions = "queryallactions"
  OpReadAllProperties = "readallproperties"
  OpReadProperty = "readproperty"
  OpSubscribeAllEvents = "subscribeallevents"
  OpSubscribeEvent = "subscribeevent"
  OpUnobserveAllProperties = "unobserveallproperties"
  OpUnobserveProperty = "unobserveroperty"
  OpUnsubscribeAllEvents = "unsubscribeallevents"
  OpUnsubscribeEvent = "unsubscribeevent"
  OpWriteProperty = "writeproperty"
  WoTAPIKeySecurityScheme = "APIKeySecurityScheme"
  WoTActions = "actions"
  WoTAnyURI = "https://www.w3.org/2019/wot/thing/v1"
  WoTAtContext = "@context"
  WoTAtType = "@type"
  WoTBasicSecurityScheme = "BasicSecurityScheme"
  WoTBearerSecurityScheme = "BearerSecurityScheme"
  WoTConst = "const"
  WoTCreated = "created"
  WoTDataType = "type"
  WoTDataTypeAnyURI = "anyURI"
  WoTDataTypeArray = "array"
  WoTDataTypeBool = "boolean"
  WoTDataTypeDateTime = "dateTime"
  WoTDataTypeInteger = "integer"
  WoTDataTypeNone = ""
  WoTDataTypeNumber = "number"
  WoTDataTypeObject = "object"
  WoTDataTypeString = "string"
  WoTDataTypeUnsignedInt = "unsignedInt"
  WoTDescription = "description"
  WoTDescriptions = "descriptions"
  WoTDigestSecurityScheme = "DigestSecurityScheme"
  WoTEnum = "enum"
  WoTEvents = "events"
  WoTFormat = "format"
  WoTForms = "forms"
  WoTHref = "href"
  WoTID = "id"
  WoTInput = "input"
  WoTLinks = "links"
  WoTMaxItems = "maxItems"
  WoTMaxLength = "maxLength"
  WoTMaximum = "maximum"
  WoTMinItems = "minItems"
  WoTMinLength = "minLength"
  WoTMinimum = "minimum"
  WoTModified = "modified"
  WoTNoSecurityScheme = "NoSecurityScheme"
  WoTOAuth2SecurityScheme = "OAuth2SecurityScheme"
  WoTOperation = "op"
  WoTOutput = "output"
  WoTPSKSecurityScheme = "PSKSecurityScheme"
  WoTProperties = "properties"
  WoTReadOnly = "readOnly"
  WoTRequired = "required"
  WoTSecurity = "security"
  WoTSupport = "support"
  WoTTitle = "title"
  WoTTitles = "titles"
  WoTVersion = "version"
)
// end of WoTVocab

// type: ActionClasses
// version: 0.1
// generated: 05 Feb 25 17:42 PST
// source: github.com/hiveot/hub/api/vocab/ht-action-classes.yaml
// namespace: hiveot
const (
  ActionDimmer = "hiveot:action:dimmer"
  ActionDimmerDecrement = "hiveot:action:dimmer:decrement"
  ActionDimmerIncrement = "hiveot:action:dimmer:increment"
  ActionDimmerSet = "hiveot:action:dimmer:set"
  ActionMedia = "hiveot:action:media"
  ActionMediaMute = "hiveot:action:media:mute"
  ActionMediaNext = "hiveot:action:media:next"
  ActionMediaPause = "hiveot:action:media:pause"
  ActionMediaPlay = "hiveot:action:media:play"
  ActionMediaPrevious = "hiveot:action:media:previous"
  ActionMediaUnmute = "hiveot:action:media:unmute"
  ActionMediaVolume = "hiveot:action:media:volume"
  ActionMediaVolumeDecrease = "hiveot:action:media:volume:decrease"
  ActionMediaVolumeIncrease = "hiveot:action:media:volume:increase"
  ActionSwitch = "hiveot:action:switch"
  ActionSwitchOnOff = "hiveot:action:switch:onoff"
  ActionSwitchToggle = "hiveot:action:switch:toggle"
  ActionThingDisable = "hiveot:action:thing:disable"
  ActionThingEnable = "hiveot:action:thing:enable"
  ActionThingStart = "hiveot:action:thing:start"
  ActionThingStop = "hiveot:action:thing:stop"
  ActionValveClose = "hiveot:action:valve:close"
  ActionValveOpen = "hiveot:action:valve:open"
)
// end of ActionClasses

// ActionClassesMap maps @type to symbol, title and description
var ActionClassesMap = map[string]struct {
   Symbol string; Title string; Description string
} {
  ActionMediaNext: {Symbol: "", Title: "Next", Description: "Next track or station"},
  ActionMediaPause: {Symbol: "", Title: "Pause", Description: "Pause playback"},
  ActionSwitchOnOff: {Symbol: "", Title: "Set On/Off switch", Description: "Action to set the switch on/off state"},
  ActionThingDisable: {Symbol: "", Title: "Disable", Description: "Action to disable a thing"},
  ActionThingEnable: {Symbol: "", Title: "Enable", Description: "Action to enable a thing"},
  ActionMedia: {Symbol: "", Title: "Media control", Description: "Commands to control media recording and playback"},
  ActionMediaMute: {Symbol: "", Title: "Mute", Description: "Mute audio"},
  ActionMediaPlay: {Symbol: "", Title: "Play", Description: "Start or continue playback"},
  ActionMediaPrevious: {Symbol: "", Title: "Previous", Description: "Previous track or station"},
  ActionMediaUnmute: {Symbol: "", Title: "Unmute", Description: "Unmute audio"},
  ActionMediaVolume: {Symbol: "", Title: "Volume", Description: "Set volume level"},
  ActionMediaVolumeIncrease: {Symbol: "", Title: "Increase volume", Description: "Increase volume"},
  ActionDimmerSet: {Symbol: "", Title: "Set dimmer", Description: "Action to set the dimmer value"},
  ActionValveClose: {Symbol: "", Title: "Close valve", Description: "Action to close the valve"},
  ActionValveOpen: {Symbol: "", Title: "Open valve", Description: "Action to open the valve"},
  ActionMediaVolumeDecrease: {Symbol: "", Title: "Decrease volume", Description: "Decrease volume"},
  ActionDimmer: {Symbol: "", Title: "Dimmer", Description: "General dimmer action"},
  ActionDimmerDecrement: {Symbol: "", Title: "Lower dimmer", Description: ""},
  ActionDimmerIncrement: {Symbol: "", Title: "Increase dimmer", Description: ""},
  ActionSwitch: {Symbol: "", Title: "Switch", Description: "General switch action"},
  ActionSwitchToggle: {Symbol: "", Title: "Toggle switch", Description: "Action to toggle the switch"},
  ActionThingStart: {Symbol: "", Title: "Start", Description: "Start running a task"},
  ActionThingStop: {Symbol: "", Title: "Stop", Description: "Stop a running task"},
}


// type: PropertyClasses
// version: 0.1
// generated: 05 Feb 25 17:42 PST
// source: github.com/hiveot/hub/api/vocab/ht-property-classes.yaml
// namespace: hiveot
const (
  PropAlarmMotion = "hiveot:prop:alarm:motion"
  PropAlarmStatus = "hiveot:prop:alarm:status"
  PropDevice = "hiveot:prop:device"
  PropDeviceBattery = "hiveot:prop:device:battery"
  PropDeviceDescription = "hiveot:prop:device:description"
  PropDeviceEnabledDisabled = "hiveot:prop:device:enabled-disabled"
  PropDeviceFirmwareVersion = "hiveot:prop:device:firmwareversion"
  PropDeviceHardwareVersion = "hiveot:prop:device:hardwareversion"
  PropDeviceMake = "hiveot:prop:device:make"
  PropDeviceModel = "hiveot:prop:device:model"
  PropDevicePollinterval = "hiveot:prop:device:pollinterval"
  PropDeviceSoftwareVersion = "hiveot:prop:device:softwareversion"
  PropDeviceStatus = "hiveot:prop:device:status"
  PropDeviceTitle = "hiveot:prop:device:title"
  PropElectric = "hiveot:prop:electric"
  PropElectricCurrent = "hiveot:prop:electric:current"
  PropElectricEnergy = "hiveot:prop:electric:energy"
  PropElectricOverload = "hiveot:prop:electric:overload"
  PropElectricPower = "hiveot:prop:electric:poer"
  PropElectricVoltage = "hiveot:prop:electric:voltage"
  PropEnv = "hiveot:prop:env"
  PropEnvAcceleration = "hiveot:prop:env:acceleration"
  PropEnvAirquality = "hiveot:prop:env:airquality"
  PropEnvBarometer = "hiveot:prop:env:barometer"
  PropEnvCO = "hiveot:prop:env:co"
  PropEnvCO2 = "hiveot:prop:env:co2"
  PropEnvCpuload = "hiveot:prop:env:cpuload"
  PropEnvDewpoint = "hiveot:prop:env:dewpoint"
  PropEnvFuelFlowrate = "hiveot:prop:env:fuel:flowrate"
  PropEnvFuelLevel = "hiveot:prop:env:fuel:level"
  PropEnvHumidex = "hiveot:prop:env:humidex"
  PropEnvHumidity = "hiveot:prop:env:humidity"
  PropEnvLuminance = "hiveot:prop:env:luminance"
  PropEnvPressure = "hiveot:prop:env:pressure"
  PropEnvTemperature = "hiveot:prop:env:temperature"
  PropEnvTimezone = "hiveot:prop:env:timezone"
  PropEnvUV = "hiveot:prop:env:uv"
  PropEnvVibration = "hiveot:prop:env:vibration"
  PropEnvVolume = "hiveot:prop:env:volume"
  PropEnvWaterFlowrate = "hiveot:prop:env:water:flowrate"
  PropEnvWaterLevel = "hiveot:prop:env:water:level"
  PropEnvWindHeading = "hiveot:prop:env:wind:heading"
  PropEnvWindSpeed = "hiveot:prop:env:wind:speed"
  PropLocation = "hiveot:prop:location"
  PropLocationCity = "hiveot:prop:location:city"
  PropLocationLatitude = "hiveot:prop:location:latitude"
  PropLocationLongitude = "hiveot:prop:location:longitude"
  PropLocationName = "hiveot:prop:location:name"
  PropLocationStreet = "hiveot:prop:location:street"
  PropLocationZipcode = "hiveot:prop:location:zipcode"
  PropMedia = "hiveot:prop:media"
  PropMediaMuted = "hiveot:prop:media:muted"
  PropMediaPaused = "hiveot:prop:media:paused"
  PropMediaPlaying = "hiveot:prop:media:playing"
  PropMediaStation = "hiveot:prop:media:station"
  PropMediaTrack = "hiveot:prop:media:track"
  PropMediaVolume = "hiveot:prop:media:volume"
  PropNet = "hiveot:prop:net"
  PropNetAddress = "hiveot:prop:net:address"
  PropNetConnection = "hiveot:prop:net:connection"
  PropNetDomainname = "hiveot:prop:net:domainname"
  PropNetGateway = "hiveot:prop:net:gateway"
  PropNetHostname = "hiveot:prop:net:hostname"
  PropNetIP4 = "hiveot:prop:net:ip4"
  PropNetIP6 = "hiveot:prop:net:ip6"
  PropNetLatency = "hiveot:prop:net:latency"
  PropNetMAC = "hiveot:prop:net:mac"
  PropNetMask = "hiveot:prop:net:mask"
  PropNetPort = "hiveot:prop:net:port"
  PropNetSignalstrength = "hiveot:prop:net:signalstrength"
  PropNetSubnet = "hiveot:prop:net:subnet"
  PropStatusOnOff = "hiveot:prop:status:onoff"
  PropStatusOpenClosed = "hiveot:prop:status:openclosed"
  PropStatusStartedStopped = "hiveot:prop:status:started-stopped"
  PropStatusYesNo = "hiveot:prop:status:yes-no"
  PropSwitch = "hiveot:prop:switch"
  PropSwitchDimmer = "hiveot:prop:switch:dimmer"
  PropSwitchLight = "hiveot:prop:switch:light"
  PropSwitchLocked = "hiveot:prop:switch:locked"
  PropSwitchOnOff = "hiveot:prop:switch:onoff"
)
// end of PropertyClasses

// PropertyClassesMap maps @type to symbol, title and description
var PropertyClassesMap = map[string]struct {
   Symbol string; Title string; Description string
} {
  PropSwitchOnOff: {Symbol: "", Title: "On/Off switch", Description: ""},
  PropMediaVolume: {Symbol: "", Title: "Volume", Description: "Media volume setting"},
  PropNetIP4: {Symbol: "", Title: "IP4 address", Description: "Device IP4 address"},
  PropDeviceDescription: {Symbol: "", Title: "Description", Description: "Device product description"},
  PropLocationZipcode: {Symbol: "", Title: "Zip code", Description: "Location ZIP code"},
  PropMediaPlaying: {Symbol: "", Title: "Playing", Description: "Media is playing"},
  PropMediaStation: {Symbol: "", Title: "Station", Description: "Selected radio station"},
  PropStatusOnOff: {Symbol: "", Title: "On/off status", Description: ""},
  PropEnvUV: {Symbol: "", Title: "UV", Description: ""},
  PropLocationName: {Symbol: "", Title: "Location name", Description: "Name of the location"},
  PropEnvPressure: {Symbol: "", Title: "Pressure", Description: ""},
  PropEnvTimezone: {Symbol: "", Title: "Timezone", Description: ""},
  PropEnvVibration: {Symbol: "", Title: "Vibration", Description: ""},
  PropLocationStreet: {Symbol: "", Title: "Street", Description: "Street address"},
  PropNet: {Symbol: "", Title: "Network properties", Description: "General network properties"},
  PropNetMAC: {Symbol: "", Title: "MAC", Description: "Hardware MAC address"},
  PropAlarmStatus: {Symbol: "", Title: "Alarm state", Description: "Current alarm status"},
  PropEnvHumidex: {Symbol: "", Title: "Humidex", Description: ""},
  PropNetSignalstrength: {Symbol: "", Title: "Signal strength", Description: "Wireless signal strength"},
  PropLocationLongitude: {Symbol: "", Title: "Longitude", Description: "Longitude geographic coordinate"},
  PropSwitchDimmer: {Symbol: "", Title: "Dimmer value", Description: ""},
  PropEnvBarometer: {Symbol: "", Title: "Atmospheric pressure", Description: "Barometric pressure of the atmosphere"},
  PropEnvFuelFlowrate: {Symbol: "", Title: "Fuel flow rate", Description: ""},
  PropElectricPower: {Symbol: "", Title: "Power", Description: "Electrical power being consumed"},
  PropEnv: {Symbol: "", Title: "Environmental property", Description: "Property of environmental sensor"},
  PropEnvAcceleration: {Symbol: "", Title: "Acceleration", Description: ""},
  PropNetHostname: {Symbol: "", Title: "Hostname", Description: "Hostname of the client"},
  PropSwitch: {Symbol: "", Title: "Switch status", Description: ""},
  PropDeviceHardwareVersion: {Symbol: "", Title: "Hardware version", Description: ""},
  PropDeviceSoftwareVersion: {Symbol: "", Title: "Software version", Description: ""},
  PropNetConnection: {Symbol: "", Title: "Connection", Description: "Connection status, connected, connecting, retrying, disconnected,..."},
  PropNetMask: {Symbol: "", Title: "Netmask", Description: "Network mask. Example: 255.255.255.0 or 24/8"},
  PropStatusStartedStopped: {Symbol: "", Title: "Started/Stopped", Description: "Started or stopped status"},
  PropSwitchLight: {Symbol: "", Title: "Light switch", Description: ""},
  PropDeviceBattery: {Symbol: "", Title: "Battery level", Description: "Device battery level"},
  PropDeviceStatus: {Symbol: "", Title: "Status", Description: "Device status; alive, awake, dead, sleeping"},
  PropEnvCO2: {Symbol: "", Title: "Carbon dioxide level", Description: "Carbon dioxide level"},
  PropNetLatency: {Symbol: "", Title: "Network latency", Description: "Delay between hub and client"},
  PropNetPort: {Symbol: "", Title: "Port", Description: "Network port"},
  PropStatusOpenClosed: {Symbol: "", Title: "Open/Closed status", Description: ""},
  PropDeviceMake: {Symbol: "", Title: "Make", Description: "Device manufacturer"},
  PropElectricOverload: {Symbol: "", Title: "Overload protection", Description: "Cut load on overload"},
  PropDevicePollinterval: {Symbol: "", Title: "Polling interval", Description: "Interval to poll for updates"},
  PropEnvHumidity: {Symbol: "", Title: "Humidity", Description: ""},
  PropEnvLuminance: {Symbol: "", Title: "Luminance", Description: ""},
  PropEnvWaterLevel: {Symbol: "", Title: "Water level", Description: ""},
  PropEnvWindSpeed: {Symbol: "", Title: "Wind speed", Description: ""},
  PropLocationLatitude: {Symbol: "", Title: "Latitude", Description: "Latitude geographic coordinate"},
  PropDeviceEnabledDisabled: {Symbol: "", Title: "Enabled/Disabled", Description: "Enabled or disabled state"},
  PropDeviceModel: {Symbol: "", Title: "Model", Description: "Device model"},
  PropEnvWindHeading: {Symbol: "", Title: "Wind heading", Description: ""},
  PropMediaPaused: {Symbol: "", Title: "Paused", Description: "Media is paused"},
  PropEnvAirquality: {Symbol: "", Title: "Air quality", Description: "Air quality level"},
  PropEnvVolume: {Symbol: "", Title: "Volume", Description: ""},
  PropNetDomainname: {Symbol: "", Title: "Domain name", Description: "Domainname of the client"},
  PropNetSubnet: {Symbol: "", Title: "Subnet", Description: "Network subnet address. Example: 192.168.0.0"},
  PropDeviceFirmwareVersion: {Symbol: "", Title: "Firmware version", Description: ""},
  PropElectric: {Symbol: "", Title: "Electrical properties", Description: "General group of electrical properties"},
  PropSwitchLocked: {Symbol: "", Title: "Lock", Description: "Electric lock status"},
  PropElectricEnergy: {Symbol: "", Title: "Energy", Description: "Electrical energy consumed"},
  PropStatusYesNo: {Symbol: "", Title: "Yes/No", Description: "Status with yes or no value"},
  PropLocation: {Symbol: "", Title: "Location", Description: "General location information"},
  PropEnvFuelLevel: {Symbol: "", Title: "Fuel level", Description: ""},
  PropEnvTemperature: {Symbol: "", Title: "Temperature", Description: ""},
  PropLocationCity: {Symbol: "", Title: "City", Description: "City name"},
  PropMedia: {Symbol: "", Title: "Media commands", Description: "Control of media equipment"},
  PropMediaMuted: {Symbol: "", Title: "Muted", Description: "Audio is muted"},
  PropNetGateway: {Symbol: "", Title: "Gateway", Description: "Network gateway address"},
  PropElectricVoltage: {Symbol: "", Title: "Voltage", Description: "Electrical voltage potential"},
  PropEnvWaterFlowrate: {Symbol: "", Title: "Water flow rate", Description: ""},
  PropEnvDewpoint: {Symbol: "", Title: "Dew point", Description: "Dew point temperature"},
  PropDeviceTitle: {Symbol: "", Title: "Title", Description: "Device friendly title"},
  PropElectricCurrent: {Symbol: "", Title: "Current", Description: "Electrical current"},
  PropEnvCO: {Symbol: "", Title: "Carbon monoxide level", Description: "Carbon monoxide level"},
  PropEnvCpuload: {Symbol: "", Title: "CPU load level", Description: "Device CPU load level"},
  PropMediaTrack: {Symbol: "", Title: "Track", Description: "Selected A/V track"},
  PropNetAddress: {Symbol: "", Title: "Address", Description: "Network address"},
  PropAlarmMotion: {Symbol: "", Title: "Motion", Description: "Motion detected"},
  PropDevice: {Symbol: "", Title: "Device attributes", Description: "Attributes describing a device"},
  PropNetIP6: {Symbol: "", Title: "IP6 address", Description: "Device IP6 address"},
}


// type: ThingClasses
// version: 0.1
// generated: 05 Feb 25 17:42 PST
// source: github.com/hiveot/hub/api/vocab/ht-thing-classes.yaml
// namespace: hiveot
const (
  ThingActuator = "hiveot:thing:actuator"
  ThingActuatorAlarm = "hiveot:thing:actuator:alarm"
  ThingActuatorBeacon = "hiveot:thing:actuator:beacon"
  ThingActuatorDimmer = "hiveot:thing:actuator:dimmer"
  ThingActuatorLight = "hiveot:thing:actuator:light"
  ThingActuatorLock = "hiveot:thing:actuator:lock"
  ThingActuatorMotor = "hiveot:thing:actuator:motor"
  ThingActuatorOutput = "hiveot:thing:actuator:output"
  ThingActuatorRanged = "hiveot:thing:actuator:ranged"
  ThingActuatorRelay = "hiveot:thing:actuator:relay"
  ThingActuatorSwitch = "hiveot:thing:actuator:switch"
  ThingActuatorValve = "hiveot:thing:actuator:valve"
  ThingActuatorValveFuel = "hiveot:thing:actuator:valve:fuel"
  ThingActuatorValveWater = "hiveot:thing:actuator:valve:water"
  ThingAppliance = "hiveot:thing:appliance"
  ThingApplianceDishwasher = "hiveot:thing:appliance:dishwasher"
  ThingApplianceDryer = "hiveot:thing:appliance:dryer"
  ThingApplianceFreezer = "hiveot:thing:appliance:freezer"
  ThingApplianceFridge = "hiveot:thing:appliance:fridge"
  ThingApplianceWasher = "hiveot:thing:appliance:washer"
  ThingComputer = "hiveot:thing:computer"
  ThingComputerCellphone = "hiveot:thing:computer:cellphone"
  ThingComputerEmbedded = "hiveot:thing:computer:embedded"
  ThingComputerMemory = "hiveot:thing:computer:memory"
  ThingComputerPC = "hiveot:thing:computer:pc"
  ThingComputerPotsPhone = "hiveot:thing:computer:potsphone"
  ThingComputerSatPhone = "hiveot:thing:computer:satphone"
  ThingComputerTablet = "hiveot:thing:computer:tablet"
  ThingComputerVoipPhone = "hiveot:thing:computer:voipphone"
  ThingControl = "hiveot:thing:control"
  ThingControlClimate = "hiveot:thing:control:climate"
  ThingControlDimmer = "hiveot:thing:control:dimmer"
  ThingControlIrrigation = "hiveot:thing:control:irrigation"
  ThingControlJoystick = "hiveot:thing:control:joystick"
  ThingControlKeypad = "hiveot:thing:control:keypad"
  ThingControlPool = "hiveot:thing:control:pool"
  ThingControlPushbutton = "hiveot:thing:control:pushbutton"
  ThingControlSwitch = "hiveot:thing:control:switch"
  ThingControlThermostat = "hiveot:thing:control:thermostat"
  ThingControlToggle = "hiveot:thing:control:toggle"
  ThingDevice = "hiveot:thing:device"
  ThingDeviceBatteryMonitor = "hiveot:thing:device:battery:monitor"
  ThingDeviceIndicator = "hiveot:thing:device:indicator"
  ThingDeviceTime = "hiveot:thing:device:time"
  ThingMedia = "hiveot:thing:media"
  ThingMediaAmplifier = "hiveot:thing:media:amplifier"
  ThingMediaCamera = "hiveot:thing:media:camera"
  ThingMediaMicrophone = "hiveot:thing:media:microphone"
  ThingMediaPlayer = "hiveot:thing:media:player"
  ThingMediaRadio = "hiveot:thing:media:radio"
  ThingMediaReceiver = "hiveot:thing:media:receiver"
  ThingMediaSpeaker = "hiveot:thing:media:speaker"
  ThingMediaTV = "hiveot:thing:media:tv"
  ThingMeter = "hiveot:thing:meter"
  ThingMeterElectric = "hiveot:thing:meter:electric"
  ThingMeterElectricCurrent = "hiveot:thing:meter:electric:current"
  ThingMeterElectricEnergy = "hiveot:thing:meter:electric:energy"
  ThingMeterElectricPower = "hiveot:thing:meter:electric:power"
  ThingMeterElectricVoltage = "hiveot:thing:meter:electric:voltage"
  ThingMeterFuel = "hiveot:thing:meter:fuel"
  ThingMeterFuelFlow = "hiveot:thing:meter:fuel:flow"
  ThingMeterFuelLevel = "hiveot:thing:meter:fuel:level"
  ThingMeterWater = "hiveot:thing:meter:water"
  ThingMeterWaterConsumption = "hiveot:thing:meter:water:consumption"
  ThingMeterWaterFlow = "hiveot:thing:meter:water:flow"
  ThingMeterWaterLevel = "hiveot:thing:meter:water:level"
  ThingMeterWind = "hiveot:thing:meter:wind"
  ThingNet = "hiveot:thing:net"
  ThingNetBluetooth = "hiveot:thing:net:bluetooth"
  ThingNetGateway = "hiveot:thing:net:gateway"
  ThingNetGatewayCoap = "hiveot:thing:net:gateway:coap"
  ThingNetGatewayInsteon = "hiveot:thing:net:gateway:insteon"
  ThingNetGatewayOnewire = "hiveot:thing:net:gateway:onewire"
  ThingNetGatewayZigbee = "hiveot:thing:net:gateway:zigbee"
  ThingNetGatewayZwave = "hiveot:thing:net:gateway:zwave"
  ThingNetLora = "hiveot:thing:net:lora"
  ThingNetLoraGateway = "hiveot:thing:net:lora:gw"
  ThingNetLoraP2P = "hiveot:thing:net:lora:p2p"
  ThingNetRouter = "hiveot:thing:net:router"
  ThingNetSwitch = "hiveot:thing:net:switch"
  ThingNetWifi = "hiveot:thing:net:wifi"
  ThingNetWifiAp = "hiveot:thing:net:wifi:ap"
  ThingSensor = "hiveot:thing:sensor"
  ThingSensorEnvironment = "hiveot:thing:sensor:environment"
  ThingSensorInput = "hiveot:thing:sensor:input"
  ThingSensorMulti = "hiveot:thing:sensor:multi"
  ThingSensorScale = "hiveot:thing:sensor:scale"
  ThingSensorSecurity = "hiveot:thing:sensor:security"
  ThingSensorSecurityDoorWindow = "hiveot:thing:sensor:security:doorwindow"
  ThingSensorSecurityGlass = "hiveot:thing:sensor:security:glass"
  ThingSensorSecurityMotion = "hiveot:thing:sensor:security:motion"
  ThingSensorSmoke = "hiveot:thing:sensor:smoke"
  ThingSensorSound = "hiveot:thing:sensor:sound"
  ThingSensorThermometer = "hiveot:thing:sensor:thermometer"
  ThingSensorWaterLeak = "hiveot:thing:sensor:water:leak"
  ThingService = "hiveot:thing:service"
)
// end of ThingClasses

// ThingClassesMap maps @type to symbol, title and description
var ThingClassesMap = map[string]struct {
   Symbol string; Title string; Description string
} {
  ThingDeviceIndicator: {Symbol: "", Title: "Indicator", Description: "Visual or audio indicator device"},
  ThingMeterFuel: {Symbol: "", Title: "Fuel metering device", Description: "General fuel metering device"},
  ThingNetLoraP2P: {Symbol: "", Title: "LoRa P2P", Description: "LoRa Peer-to-peer network device"},
  ThingSensorEnvironment: {Symbol: "", Title: "Environmental sensor", Description: "Environmental sensor with one or more features such as temperature, humidity, etc"},
  ThingSensorInput: {Symbol: "", Title: "Input sensor", Description: "General purpose electrical input sensor"},
  ThingSensorWaterLeak: {Symbol: "", Title: "Water leak detector", Description: "Dedicated water leak detector"},
  ThingActuatorOutput: {Symbol: "", Title: "Output", Description: "General purpose electrical output signal"},
  ThingMediaMicrophone: {Symbol: "", Title: "Microphone", Description: "Microphone for capturing audio"},
  ThingSensorScale: {Symbol: "", Title: "Scale", Description: "Electronic weigh scale"},
  ThingActuatorDimmer: {Symbol: "", Title: "Dimmer", Description: "Light dimmer"},
  ThingControlDimmer: {Symbol: "", Title: "Dimmer", Description: "Light dimmer input device"},
  ThingDeviceTime: {Symbol: "", Title: "Clock", Description: "Time tracking device such as clocks and time chips"},
  ThingMeterElectricVoltage: {Symbol: "", Title: "Voltage", Description: "Electrical voltage meter"},
  ThingNetGatewayZwave: {Symbol: "", Title: "ZWave gateway", Description: "Gateway providing access to ZWave devices"},
  ThingActuatorLight: {Symbol: "", Title: "Light", Description: "Smart LED or other light"},
  ThingApplianceDryer: {Symbol: "", Title: "Dryer", Description: "Clothing dryer"},
  ThingControl: {Symbol: "", Title: "Input controller", Description: "Generic input controller"},
  ThingControlPool: {Symbol: "", Title: "Pool control", Description: "Device for controlling pool settings"},
  ThingControlPushbutton: {Symbol: "", Title: "Momentary switch", Description: "Momentary push button control input"},
  ThingControlThermostat: {Symbol: "", Title: "Thermostat", Description: "Thermostat HVAC control"},
  ThingNetGatewayOnewire: {Symbol: "", Title: "1-Wire gateway", Description: "Gateway providing access to 1-wire devices"},
  ThingSensorSecurity: {Symbol: "", Title: "Security", Description: "Generic security sensor"},
  ThingActuatorMotor: {Symbol: "", Title: "Motor", Description: "Motor driven actuator, such as garage door, blinds, tv lifts"},
  ThingComputerCellphone: {Symbol: "", Title: "Cell Phone", Description: "Cellular phone"},
  ThingMedia: {Symbol: "", Title: "A/V media", Description: "Generic device for audio/video media record or playback"},
  ThingMediaRadio: {Symbol: "", Title: "Radio", Description: "AM or FM radio receiver"},
  ThingMeterFuelFlow: {Symbol: "", Title: "Fuel flow rate", Description: "Dedicated fuel flow rate metering device"},
  ThingMeterWind: {Symbol: "", Title: "Wind", Description: "Dedicated wind meter"},
  ThingNetWifiAp: {Symbol: "", Title: "Wifi access point", Description: "Wireless access point for IP networks"},
  ThingSensorSecurityGlass: {Symbol: "", Title: "Glass sensor", Description: "Dedicated sensor for detecting breaking of glass"},
  ThingDevice: {Symbol: "", Title: "Device", Description: "Device of unknown purpose"},
  ThingMediaReceiver: {Symbol: "", Title: "Receiver", Description: "Audio/video receiver and player"},
  ThingMediaTV: {Symbol: "", Title: "TV", Description: "Network connected television"},
  ThingNetGatewayInsteon: {Symbol: "", Title: "Insteon gateway", Description: "Gateway providing access to Insteon devices"},
  ThingSensorMulti: {Symbol: "", Title: "Multi sensor", Description: "Sense multiple inputs"},
  ThingActuator: {Symbol: "", Title: "Actuator", Description: "Generic actuator"},
  ThingActuatorRanged: {Symbol: "", Title: "Ranged actuator", Description: "Generic ranged actuator with a set point"},
  ThingComputerEmbedded: {Symbol: "", Title: "Embedded System", Description: "Embedded computing device"},
  ThingMeter: {Symbol: "", Title: "Meter", Description: "General metering device"},
  ThingMeterElectricCurrent: {Symbol: "", Title: "Electric current", Description: "Electrical current meter"},
  ThingMeterWaterConsumption: {Symbol: "", Title: "Water consumption meter", Description: "Water consumption meter"},
  ThingNetLora: {Symbol: "", Title: "LoRa network device", Description: "Generic Long Range network protocol device"},
  ThingApplianceWasher: {Symbol: "", Title: "Washer", Description: "Clothing washer"},
  ThingControlKeypad: {Symbol: "", Title: "Keypad", Description: "Multi-key pad for command input"},
  ThingDeviceBatteryMonitor: {Symbol: "", Title: "Battery Monitor", Description: "Battery monitor and charge controller"},
  ThingMeterElectric: {Symbol: "", Title: "", Description: ""},
  ThingApplianceFreezer: {Symbol: "", Title: "Freezer", Description: "Refrigerator freezer"},
  ThingComputerTablet: {Symbol: "", Title: "Tablet", Description: "Tablet computer"},
  ThingControlClimate: {Symbol: "", Title: "Climate control", Description: "Device for controlling climate of a space"},
  ThingMediaAmplifier: {Symbol: "", Title: "Audio amplifier", Description: "Audio amplifier with volume controls"},
  ThingMeterWaterLevel: {Symbol: "", Title: "Water level", Description: "Dedicated water level meter"},
  ThingSensor: {Symbol: "", Title: "Sensor", Description: "Generic sensor device"},
  ThingSensorThermometer: {Symbol: "", Title: "Thermometer", Description: "Environmental thermometer"},
  ThingActuatorValveFuel: {Symbol: "", Title: "Fuel valve", Description: "Electric powered fuel valve"},
  ThingComputerMemory: {Symbol: "", Title: "Memory", Description: "Stand-alone memory device such as eeprom or iButtons"},
  ThingComputerSatPhone: {Symbol: "", Title: "Satellite phone", Description: ""},
  ThingControlIrrigation: {Symbol: "", Title: "Irrigation control", Description: "Device for control of an irrigation system"},
  ThingMediaCamera: {Symbol: "", Title: "Camera", Description: "Video camera"},
  ThingMeterElectricPower: {Symbol: "", Title: "Electrical Power", Description: "Electrical power meter"},
  ThingSensorSound: {Symbol: "", Title: "Sound detector", Description: ""},
  ThingActuatorLock: {Symbol: "", Title: "Lock", Description: "Electronic door lock"},
  ThingActuatorRelay: {Symbol: "", Title: "Relay", Description: "Generic relay electrical switch"},
  ThingControlJoystick: {Symbol: "", Title: "Joystick", Description: "Flight control stick"},
  ThingControlSwitch: {Symbol: "", Title: "Input switch", Description: "On or off switch input control"},
  ThingControlToggle: {Symbol: "", Title: "Toggle switch", Description: "Toggle switch input control"},
  ThingSensorSecurityDoorWindow: {Symbol: "", Title: "Door/Window sensor", Description: "Dedicated door/window opening security sensor"},
  ThingSensorSmoke: {Symbol: "", Title: "Smoke detector", Description: ""},
  ThingApplianceFridge: {Symbol: "", Title: "Fridge", Description: "Refrigerator appliance"},
  ThingComputer: {Symbol: "", Title: "Computing Device", Description: "General purpose computing device"},
  ThingMeterWater: {Symbol: "", Title: "Water metering device", Description: "General water metering device"},
  ThingNetBluetooth: {Symbol: "", Title: "Bluetooth", Description: "Bluetooth radio"},
  ThingNetWifi: {Symbol: "", Title: "Wifi device", Description: "Generic wifi device"},
  ThingService: {Symbol: "", Title: "Service", Description: "General service for processing data and offering features of interest"},
  ThingAppliance: {Symbol: "", Title: "Appliance", Description: "Appliance to accomplish a particular task for occupant use"},
  ThingComputerVoipPhone: {Symbol: "", Title: "VoIP Phone", Description: "Voice over IP phone"},
  ThingMediaPlayer: {Symbol: "", Title: "Media player", Description: "CD/DVD/Blueray/USB player of recorded media"},
  ThingMeterElectricEnergy: {Symbol: "", Title: "Electric energy", Description: "Electrical energy meter"},
  ThingActuatorBeacon: {Symbol: "", Title: "Beacon", Description: "Location beacon"},
  ThingActuatorValve: {Symbol: "", Title: "Valve", Description: "Electric powered valve for fluids or gas"},
  ThingComputerPC: {Symbol: "", Title: "PC/Laptop", Description: "Personal computer/laptop"},
  ThingNetGateway: {Symbol: "", Title: "Gateway", Description: "Generic gateway device providing access to other devices"},
  ThingNetLoraGateway: {Symbol: "", Title: "LoRaWAN gateway", Description: "Gateway providing access to LoRa devices"},
  ThingSensorSecurityMotion: {Symbol: "", Title: "Motion sensor", Description: "Dedicated security sensor detecting motion"},
  ThingActuatorValveWater: {Symbol: "", Title: "Water valve", Description: "Electric powered water valve"},
  ThingApplianceDishwasher: {Symbol: "", Title: "Dishwasher", Description: "Dishwasher"},
  ThingComputerPotsPhone: {Symbol: "", Title: "Land Line", Description: "Plain Old Telephone System, aka landline"},
  ThingMeterWaterFlow: {Symbol: "", Title: "Water flow", Description: "Dedicated water flow-rate meter"},
  ThingNet: {Symbol: "", Title: "Network device", Description: "Generic network device"},
  ThingNetGatewayZigbee: {Symbol: "", Title: "Zigbee gateway", Description: "Gateway providing access to Zigbee devices"},
  ThingNetRouter: {Symbol: "", Title: "Network router", Description: "IP ThingNetwork router providing access to other IP networks"},
  ThingActuatorAlarm: {Symbol: "", Title: "Alarm", Description: "Siren or light alarm"},
  ThingActuatorSwitch: {Symbol: "", Title: "Switch", Description: "An electric powered on/off switch for powering circuits"},
  ThingMediaSpeaker: {Symbol: "", Title: "Connected speakers", Description: "Network connected speakers"},
  ThingMeterFuelLevel: {Symbol: "", Title: "Fuel level", Description: "Dedicated fuel level metering device"},
  ThingNetGatewayCoap: {Symbol: "", Title: "CoAP gateway", Description: "Gateway providing access to CoAP devices"},
  ThingNetSwitch: {Symbol: "", Title: "Network switch", Description: "Network switch to connect computer devices to the network"},
}


// type: UnitClasses
// version: 0.1
// generated: 05 Feb 25 17:42 PST
// source: github.com/hiveot/hub/api/vocab/ht-unit-classes.yaml
// namespace: hiveot
const (
  UnitAmpere = "hiveot:unit:ampere"
  UnitCandela = "hiveot:unit:candela"
  UnitCelcius = "hiveot:unit:celcius"
  UnitCount = "hiveot:unit:count"
  UnitDegree = "hiveot:unit:degree"
  UnitFahrenheit = "hiveot:unit:fahrenheit"
  UnitFoot = "hiveot:unit:foot"
  UnitGallon = "hiveot:unit:gallon"
  UnitKelvin = "hiveot:unit:kelvin"
  UnitKilogram = "hiveot:unit:kilogram"
  UnitKilometerPerHour = "hiveot:unit:kph"
  UnitKilowattHour = "hiveot:unit:kilowatthour"
  UnitLiter = "hiveot:unit:liter"
  UnitLumen = "hiveot:unit:lumen"
  UnitLux = "hiveot:unit:lux"
  UnitMercury = "hiveot:unit:mercury"
  UnitMeter = "hiveot:unit:meter"
  UnitMeterPerSecond = "hiveot:unit:meterspersecond"
  UnitMilesPerHour = "hiveot:unit:milesperhour"
  UnitMilliSecond = "hiveot:unit:millisecond"
  UnitMillibar = "hiveot:unit:millibar"
  UnitMole = "hiveot:unit:mole"
  UnitPSI = "hiveot:unit:psi"
  UnitPascal = "hiveot:unit:pascal"
  UnitPercent = "hiveot:unit:percent"
  UnitPound = "hiveot:unit:pound"
  UnitPpm = "hiveot:unit:ppm"
  UnitRadian = "hiveot:unit:radian"
  UnitSecond = "hiveot:unit:second"
  UnitVolt = "hiveot:unit:volt"
  UnitWatt = "hiveot:unit:watt"
)
// end of UnitClasses

// UnitClassesMap maps @type to symbol, title and description
var UnitClassesMap = map[string]struct {
   Symbol string; Title string; Description string
} {
  UnitKilowattHour: {Symbol: "kWh", Title: "Kilowatt-hour", Description: "non-SI unit of energy equivalent to 3.6 megajoules."},
  UnitLux: {Symbol: "lx", Title: "Lux", Description: "SI unit illuminance. Equal to 1 lumen per square meter."},
  UnitMillibar: {Symbol: "mbar", Title: "millibar", Description: "Metric unit of pressure. 1/1000th of a bar. Equal to 100 pascals. Amount of force it takes to move an object weighing a gram, one centimeter in one second."},
  UnitKilogram: {Symbol: "kg", Title: "Kilogram", Description: ""},
  UnitLumen: {Symbol: "lm", Title: "Lumen", Description: "SI unit luminous flux. Measure of perceived power of visible light. 1lm = 1 cd steradian"},
  UnitMilesPerHour: {Symbol: "mph", Title: "Miles per hour", Description: "Speed in miles per hour"},
  UnitGallon: {Symbol: "gl", Title: "Gallon", Description: "Unit of volume. 1 Imperial gallon is 4.54609 liters. 1 US liquid gallon is 3.78541 liters. 1 US dry gallon is 4.405 liters. "},
  UnitMeterPerSecond: {Symbol: "m/s", Title: "Meters per second", Description: "SI unit of speed in meters per second"},
  UnitPercent: {Symbol: "%", Title: "Percent", Description: "Fractions of 100"},
  UnitPpm: {Symbol: "ppm", Title: "PPM", Description: "Parts per million"},
  UnitWatt: {Symbol: "W", Title: "Watt", Description: "SI unit of power. Equal to 1 joule per second; or work performed when a current of 1 ampere flows across an electric potential of one volt."},
  UnitMole: {Symbol: "mol", Title: "Mole", Description: "SI unit of measurement for amount of substance. Eg, molecules."},
  UnitPSI: {Symbol: "PSI", Title: "PSI", Description: "Unit of pressure. Pounds of force per square inch. 1PSI equals 6984 Pascals."},
  UnitKelvin: {Symbol: "K", Title: "Kelvin", Description: "SI unit of thermodynamic temperature. 0 K represents absolute zero, the absence of all heat. 0 C equals +273.15K"},
  UnitPascal: {Symbol: "Pa", Title: "Pascal", Description: "SI unit of pressure. Equal to 1 newton of force applied over 1 square meter."},
  UnitVolt: {Symbol: "V", Title: "Volt", Description: "SI unit of electric potential; Energy consumption of 1 joule per electric charge of one coulomb"},
  UnitFoot: {Symbol: "ft", Title: "Foot", Description: "Imperial unit of distance. 1 foot equals 0.3048 meters"},
  UnitLiter: {Symbol: "l", Title: "Liter", Description: "SI unit of volume equivalent to 1 cubic decimeter."},
  UnitCelcius: {Symbol: "°C", Title: "Celcius", Description: "Temperature in Celcius"},
  UnitRadian: {Symbol: "", Title: "Radian", Description: "Angle in 0-2pi"},
  UnitSecond: {Symbol: "s", Title: "Second", Description: "SI unit of time based on caesium frequency"},
  UnitAmpere: {Symbol: "A", Title: "Ampere", Description: "Electrical current in Amperes based on the elementary charge flow per second"},
  UnitCandela: {Symbol: "cd", Title: "Candela", Description: "SI unit of luminous intensity in a given direction. Roughly the same brightness as the common candle."},
  UnitCount: {Symbol: "(N)", Title: "Count", Description: ""},
  UnitDegree: {Symbol: "degree", Title: "Degree", Description: "Angle in 0-360 degrees"},
  UnitFahrenheit: {Symbol: "F", Title: "Fahrenheit", Description: "Temperature in Fahrenheit"},
  UnitKilometerPerHour: {Symbol: "kph", Title: "Km per hour", Description: "Speed in kilometers per hour"},
  UnitMercury: {Symbol: "Hg", Title: "Mercury", Description: "Unit of atmospheric pressure in the United States. 1 Hg equals 33.8639 mbar."},
  UnitMeter: {Symbol: "m", Title: "Meter", Description: "Distance in meters. 1m=c/299792458"},
  UnitMilliSecond: {Symbol: "ms", Title: "millisecond", Description: "Unit of time in milli-seconds. Equal to 1/1000 of a second."},
  UnitPound: {Symbol: "lbs", Title: "Pound", Description: "Imperial unit of weight. Equivalent to 0.453592 Kg. 1 Kg is 2.205 lbs"},
}
