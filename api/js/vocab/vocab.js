// Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions
// DO NOT EDIT. This file is generated and changes will be overwritten

// type: ActionStatusStatus
// version: 0.1
// generated: 19 Nov 24 19:05 PST
// source: github.com/hiveot/hub/api/vocab/ht-constants.yaml
// description: Request progress status constants
export const RequestCompleted = "completed"
export const RequestDelivered = "delivered"
export const RequestFailed = "failed"
export const RequestPending = "pending"
// end of ActionStatusStatus

// type: WoTVocab
// version: 0.1
// generated: 19 Nov 24 19:05 PST
// source: github.com/hiveot/hub/api/vocab/wot-vocab.yaml
// description: WoT vocabulary definition. See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
export const HTOpLogin = "login"
export const HTOpLogout = "logout"
export const HTOpPublishError = "error"
export const HTOpPublishEvent = "event"
export const HTOpReadAllEvents = "readallevents"
export const HTOpReadAllTDs = "readalltds"
export const HTOpReadEvent = "readevent"
export const HTOpReadTD = "readtd"
export const HTOpRefresh = "refresh"
export const HTOpUpdateActionStatus = "actionstatus"
export const HTOpUpdateActionStatuses = "actionstatuses"
export const HTOpUpdateMultipleProperties = "updatemultipleproperties"
export const HTOpUpdateProperties = "updateproperties"
export const HTOpUpdateProperty = "updateproperty"
export const HTOpUpdateTD = "updatetd"
export const OpCancelAction = "cancelaction"
export const OpInvokeAction = "invokeaction"
export const OpObserveAllProperties = "observeallproperties"
export const OpObserveProperty = "observeproperty"
export const OpQueryAction = "queryaction"
export const OpQueryAllActions = "queryallactions"
export const OpReadAllProperties = "readallproperties"
export const OpReadMultipleProperties = "readmultipleproperties"
export const OpReadProperty = "readproperty"
export const OpSubscribeAllEvents = "subscribeallevents"
export const OpSubscribeEvent = "subscribeevent"
export const OpUnobserveAllProperties = "unobserveallproperties"
export const OpUnobserveProperty = "unobserveroperty"
export const OpUnsubscribeAllEvents = "unsubscribeallevents"
export const OpUnsubscribeEvent = "unsubscribeevent"
export const OpWriteAllProperties = "writeallproperties"
export const OpWriteMultipleProperties = "writemultipleproperties"
export const OpWriteProperty = "writeproperty"
export const WoTAPIKeySecurityScheme = "APIKeySecurityScheme"
export const WoTActions = "actions"
export const WoTAnyURI = "https://www.w3.org/2019/wot/thing/v1"
export const WoTAtContext = "@context"
export const WoTAtType = "@type"
export const WoTBasicSecurityScheme = "BasicSecurityScheme"
export const WoTBearerSecurityScheme = "BearerSecurityScheme"
export const WoTConst = "const"
export const WoTCreated = "created"
export const WoTDataType = "type"
export const WoTDataTypeAnyURI = "anyURI"
export const WoTDataTypeArray = "array"
export const WoTDataTypeBool = "boolean"
export const WoTDataTypeDateTime = "dateTime"
export const WoTDataTypeInteger = "integer"
export const WoTDataTypeNone = ""
export const WoTDataTypeNumber = "number"
export const WoTDataTypeObject = "object"
export const WoTDataTypeString = "string"
export const WoTDataTypeUnsignedInt = "unsignedInt"
export const WoTDescription = "description"
export const WoTDescriptions = "descriptions"
export const WoTDigestSecurityScheme = "DigestSecurityScheme"
export const WoTEnum = "enum"
export const WoTEvents = "events"
export const WoTFormat = "format"
export const WoTForms = "forms"
export const WoTHref = "href"
export const WoTID = "id"
export const WoTInput = "input"
export const WoTLinks = "links"
export const WoTMaxItems = "maxItems"
export const WoTMaxLength = "maxLength"
export const WoTMaximum = "maximum"
export const WoTMinItems = "minItems"
export const WoTMinLength = "minLength"
export const WoTMinimum = "minimum"
export const WoTModified = "modified"
export const WoTNoSecurityScheme = "NoSecurityScheme"
export const WoTOAuth2SecurityScheme = "OAuth2SecurityScheme"
export const WoTOperation = "op"
export const WoTOutput = "output"
export const WoTPSKSecurityScheme = "PSKSecurityScheme"
export const WoTProperties = "properties"
export const WoTReadOnly = "readOnly"
export const WoTRequired = "required"
export const WoTSecurity = "security"
export const WoTSupport = "support"
export const WoTTitle = "title"
export const WoTTitles = "titles"
export const WoTVersion = "version"
// end of WoTVocab

// type: UnitClasses
// version: 0.1
// generated: 19 Nov 24 19:05 PST
// source: github.com/hiveot/hub/api/vocab/ht-unit-classes.yaml
// namespace: ht
export const UnitAmpere = "ht:unit:ampere";
export const UnitCandela = "ht:unit:candela";
export const UnitCelcius = "ht:unit:celcius";
export const UnitCount = "ht:unit:count";
export const UnitDegree = "ht:unit:degree";
export const UnitFahrenheit = "ht:unit:fahrenheit";
export const UnitFoot = "ht:unit:foot";
export const UnitGallon = "ht:unit:gallon";
export const UnitKelvin = "ht:unit:kelvin";
export const UnitKilogram = "ht:unit:kilogram";
export const UnitKilometerPerHour = "ht:unit:kph";
export const UnitKilowattHour = "ht:unit:kilowatthour";
export const UnitLiter = "ht:unit:liter";
export const UnitLumen = "ht:unit:lumen";
export const UnitLux = "ht:unit:lux";
export const UnitMercury = "ht:unit:mercury";
export const UnitMeter = "ht:unit:meter";
export const UnitMeterPerSecond = "ht:unit:meterspersecond";
export const UnitMilesPerHour = "ht:unit:milesperhour";
export const UnitMilliSecond = "ht:unit:millisecond";
export const UnitMillibar = "ht:unit:millibar";
export const UnitMole = "ht:unit:mole";
export const UnitPSI = "ht:unit:psi";
export const UnitPascal = "ht:unit:pascal";
export const UnitPercent = "ht:unit:percent";
export const UnitPound = "ht:unit:pound";
export const UnitPpm = "ht:unit:ppm";
export const UnitRadian = "ht:unit:radian";
export const UnitSecond = "ht:unit:second";
export const UnitVolt = "ht:unit:volt";
export const UnitWatt = "ht:unit:watt";
// end of UnitClasses

// UnitClassesMap maps @type to symbol, title and description
export const UnitClassesMap = {
  "ht:unit:ppm": {Symbol: "ppm", Title: "PPM", Description: "Parts per million"},
  "ht:unit:psi": {Symbol: "PSI", Title: "PSI", Description: "Unit of pressure. Pounds of force per square inch. 1PSI equals 6984 Pascals."},
  "ht:unit:watt": {Symbol: "W", Title: "Watt", Description: "SI unit of power. Equal to 1 joule per second; or work performed when a current of 1 ampere flows across an electric potential of one volt."},
  "ht:unit:mercury": {Symbol: "Hg", Title: "Mercury", Description: "Unit of atmospheric pressure in the United States. 1 Hg equals 33.8639 mbar."},
  "ht:unit:meter": {Symbol: "m", Title: "Meter", Description: "Distance in meters. 1m=c/299792458"},
  "ht:unit:millisecond": {Symbol: "ms", Title: "millisecond", Description: "Unit of time in milli-seconds. Equal to 1/1000 of a second."},
  "ht:unit:pascal": {Symbol: "Pa", Title: "Pascal", Description: "SI unit of pressure. Equal to 1 newton of force applied over 1 square meter."},
  "ht:unit:ampere": {Symbol: "A", Title: "Ampere", Description: "Electrical current in Amperes based on the elementary charge flow per second"},
  "ht:unit:count": {Symbol: "(N)", Title: "Count", Description: ""},
  "ht:unit:kph": {Symbol: "kph", Title: "Km per hour", Description: "Speed in kilometers per hour"},
  "ht:unit:pound": {Symbol: "lbs", Title: "Pound", Description: "Imperial unit of weight. Equivalent to 0.453592 Kg. 1 Kg is 2.205 lbs"},
  "ht:unit:radian": {Symbol: "", Title: "Radian", Description: "Angle in 0-2pi"},
  "ht:unit:celcius": {Symbol: "°C", Title: "Celcius", Description: "Temperature in Celcius"},
  "ht:unit:milesperhour": {Symbol: "mph", Title: "Miles per hour", Description: "Speed in miles per hour"},
  "ht:unit:millibar": {Symbol: "mbar", Title: "millibar", Description: "Metric unit of pressure. 1/1000th of a bar. Equal to 100 pascals. Amount of force it takes to move an object weighing a gram, one centimeter in one second."},
  "ht:unit:degree": {Symbol: "degree", Title: "Degree", Description: "Angle in 0-360 degrees"},
  "ht:unit:gallon": {Symbol: "gl", Title: "Gallon", Description: "Unit of volume. 1 Imperial gallon is 4.54609 liters. 1 US liquid gallon is 3.78541 liters. 1 US dry gallon is 4.405 liters. "},
  "ht:unit:liter": {Symbol: "l", Title: "Liter", Description: "SI unit of volume equivalent to 1 cubic decimeter."},
  "ht:unit:mole": {Symbol: "mol", Title: "Mole", Description: "SI unit of measurement for amount of substance. Eg, molecules."},
  "ht:unit:lumen": {Symbol: "lm", Title: "Lumen", Description: "SI unit luminous flux. Measure of perceived power of visible light. 1lm = 1 cd steradian"},
  "ht:unit:percent": {Symbol: "%", Title: "Percent", Description: "Fractions of 100"},
  "ht:unit:volt": {Symbol: "V", Title: "Volt", Description: "SI unit of electric potential; Energy consumption of 1 joule per electric charge of one coulomb"},
  "ht:unit:fahrenheit": {Symbol: "F", Title: "Fahrenheit", Description: "Temperature in Fahrenheit"},
  "ht:unit:kelvin": {Symbol: "K", Title: "Kelvin", Description: "SI unit of thermodynamic temperature. 0 K represents absolute zero, the absence of all heat. 0 C equals +273.15K"},
  "ht:unit:meterspersecond": {Symbol: "m/s", Title: "Meters per second", Description: "SI unit of speed in meters per second"},
  "ht:unit:kilowatthour": {Symbol: "kWh", Title: "Kilowatt-hour", Description: "non-SI unit of energy equivalent to 3.6 megajoules."},
  "ht:unit:lux": {Symbol: "lx", Title: "Lux", Description: "SI unit illuminance. Equal to 1 lumen per square meter."},
  "ht:unit:second": {Symbol: "s", Title: "Second", Description: "SI unit of time based on caesium frequency"},
  "ht:unit:candela": {Symbol: "cd", Title: "Candela", Description: "SI unit of luminous intensity in a given direction. Roughly the same brightness as the common candle."},
  "ht:unit:foot": {Symbol: "ft", Title: "Foot", Description: "Imperial unit of distance. 1 foot equals 0.3048 meters"},
  "ht:unit:kilogram": {Symbol: "kg", Title: "Kilogram", Description: ""},
}


// type: ActionClasses
// version: 0.1
// generated: 19 Nov 24 19:05 PST
// source: github.com/hiveot/hub/api/vocab/ht-action-classes.yaml
// namespace: ht
export const ActionDimmer = "ht:action:dimmer";
export const ActionDimmerDecrement = "ht:action:dimmer:decrement";
export const ActionDimmerIncrement = "ht:action:dimmer:increment";
export const ActionDimmerSet = "ht:action:dimmer:set";
export const ActionMedia = "ht:action:media";
export const ActionMediaMute = "ht:action:media:mute";
export const ActionMediaNext = "ht:action:media:next";
export const ActionMediaPause = "ht:action:media:pause";
export const ActionMediaPlay = "ht:action:media:play";
export const ActionMediaPrevious = "ht:action:media:previous";
export const ActionMediaUnmute = "ht:action:media:unmute";
export const ActionMediaVolume = "ht:action:media:volume";
export const ActionMediaVolumeDecrease = "ht:action:media:volume:decrease";
export const ActionMediaVolumeIncrease = "ht:action:media:volume:increase";
export const ActionSwitch = "ht:action:switch";
export const ActionSwitchOnOff = "ht:action:switch:onoff";
export const ActionSwitchToggle = "ht:action:switch:toggle";
export const ActionThingDisable = "ht:action:thing:disable";
export const ActionThingEnable = "ht:action:thing:enable";
export const ActionThingStart = "ht:action:thing:start";
export const ActionThingStop = "ht:action:thing:stop";
export const ActionValveClose = "ht:action:valve:close";
export const ActionValveOpen = "ht:action:valve:open";
// end of ActionClasses

// ActionClassesMap maps @type to symbol, title and description
export const ActionClassesMap = {
  "ht:action:media": {Symbol: "", Title: "Media control", Description: "Commands to control media recording and playback"},
  "ht:action:dimmer:decrement": {Symbol: "", Title: "Lower dimmer", Description: ""},
  "ht:action:dimmer:set": {Symbol: "", Title: "Set dimmer", Description: "Action to set the dimmer value"},
  "ht:action:switch:toggle": {Symbol: "", Title: "Toggle switch", Description: "Action to toggle the switch"},
  "ht:action:thing:enable": {Symbol: "", Title: "Enable", Description: "Action to enable a thing"},
  "ht:action:media:unmute": {Symbol: "", Title: "Unmute", Description: "Unmute audio"},
  "ht:action:media:volume:decrease": {Symbol: "", Title: "Decrease volume", Description: "Decrease volume"},
  "ht:action:thing:disable": {Symbol: "", Title: "Disable", Description: "Action to disable a thing"},
  "ht:action:valve:close": {Symbol: "", Title: "Close valve", Description: "Action to close the valve"},
  "ht:action:media:mute": {Symbol: "", Title: "Mute", Description: "Mute audio"},
  "ht:action:media:next": {Symbol: "", Title: "Next", Description: "Next track or station"},
  "ht:action:dimmer": {Symbol: "", Title: "Dimmer", Description: "General dimmer action"},
  "ht:action:switch": {Symbol: "", Title: "Switch", Description: "General switch action"},
  "ht:action:thing:stop": {Symbol: "", Title: "Stop", Description: "Stop a running task"},
  "ht:action:switch:onoff": {Symbol: "", Title: "Set On/Off switch", Description: "Action to set the switch on/off state"},
  "ht:action:thing:start": {Symbol: "", Title: "Start", Description: "Start running a task"},
  "ht:action:media:pause": {Symbol: "", Title: "Pause", Description: "Pause playback"},
  "ht:action:media:play": {Symbol: "", Title: "Play", Description: "Start or continue playback"},
  "ht:action:media:previous": {Symbol: "", Title: "Previous", Description: "Previous track or station"},
  "ht:action:media:volume": {Symbol: "", Title: "Volume", Description: "Set volume level"},
  "ht:action:media:volume:increase": {Symbol: "", Title: "Increase volume", Description: "Increase volume"},
  "ht:action:dimmer:increment": {Symbol: "", Title: "Increase dimmer", Description: ""},
  "ht:action:valve:open": {Symbol: "", Title: "Open valve", Description: "Action to open the valve"},
}


// type: PropertyClasses
// version: 0.1
// generated: 19 Nov 24 19:05 PST
// source: github.com/hiveot/hub/api/vocab/ht-property-classes.yaml
// namespace: ht
export const PropAlarmMotion = "ht:prop:alarm:motion";
export const PropAlarmStatus = "ht:prop:alarm:status";
export const PropDevice = "ht:prop:device";
export const PropDeviceBattery = "ht:prop:device:battery";
export const PropDeviceDescription = "ht:prop:device:description";
export const PropDeviceEnabledDisabled = "ht:prop:device:enabled-disabled";
export const PropDeviceFirmwareVersion = "ht:prop:device:firmwareversion";
export const PropDeviceHardwareVersion = "ht:prop:device:hardwareversion";
export const PropDeviceMake = "ht:prop:device:make";
export const PropDeviceModel = "ht:prop:device:model";
export const PropDevicePollinterval = "ht:prop:device:pollinterval";
export const PropDeviceSoftwareVersion = "ht:prop:device:softwareversion";
export const PropDeviceStatus = "ht:prop:device:status";
export const PropDeviceTitle = "ht:prop:device:title";
export const PropElectric = "ht:prop:electric";
export const PropElectricCurrent = "ht:prop:electric:current";
export const PropElectricEnergy = "ht:prop:electric:energy";
export const PropElectricOverload = "ht:prop:electric:overload";
export const PropElectricPower = "ht:prop:electric:poer";
export const PropElectricVoltage = "ht:prop:electric:voltage";
export const PropEnv = "ht:prop:env";
export const PropEnvAcceleration = "ht:prop:env:acceleration";
export const PropEnvAirquality = "ht:prop:env:airquality";
export const PropEnvBarometer = "ht:prop:env:barometer";
export const PropEnvCO = "ht:prop:env:co";
export const PropEnvCO2 = "ht:prop:env:co2";
export const PropEnvCpuload = "ht:prop:env:cpuload";
export const PropEnvDewpoint = "ht:prop:env:dewpoint";
export const PropEnvFuelFlowrate = "ht:prop:env:fuel:flowrate";
export const PropEnvFuelLevel = "ht:prop:env:fuel:level";
export const PropEnvHumidex = "ht:prop:env:humidex";
export const PropEnvHumidity = "ht:prop:env:humidity";
export const PropEnvLuminance = "ht:prop:env:luminance";
export const PropEnvPressure = "ht:prop:env:pressure";
export const PropEnvTemperature = "ht:prop:env:temperature";
export const PropEnvTimezone = "ht:prop:env:timezone";
export const PropEnvUV = "ht:prop:env:uv";
export const PropEnvVibration = "ht:prop:env:vibration";
export const PropEnvVolume = "ht:prop:env:volume";
export const PropEnvWaterFlowrate = "ht:prop:env:water:flowrate";
export const PropEnvWaterLevel = "ht:prop:env:water:level";
export const PropEnvWindHeading = "ht:prop:env:wind:heading";
export const PropEnvWindSpeed = "ht:prop:env:wind:speed";
export const PropLocation = "ht:prop:location";
export const PropLocationCity = "ht:prop:location:city";
export const PropLocationLatitude = "ht:prop:location:latitude";
export const PropLocationLongitude = "ht:prop:location:longitude";
export const PropLocationName = "ht:prop:location:name";
export const PropLocationStreet = "ht:prop:location:street";
export const PropLocationZipcode = "ht:prop:location:zipcode";
export const PropMedia = "ht:prop:media";
export const PropMediaMuted = "ht:prop:media:muted";
export const PropMediaPaused = "ht:prop:media:paused";
export const PropMediaPlaying = "ht:prop:media:playing";
export const PropMediaStation = "ht:prop:media:station";
export const PropMediaTrack = "ht:prop:media:track";
export const PropMediaVolume = "ht:prop:media:volume";
export const PropNet = "ht:prop:net";
export const PropNetAddress = "ht:prop:net:address";
export const PropNetConnection = "ht:prop:net:connection";
export const PropNetDomainname = "ht:prop:net:domainname";
export const PropNetGateway = "ht:prop:net:gateway";
export const PropNetHostname = "ht:prop:net:hostname";
export const PropNetIP4 = "ht:prop:net:ip4";
export const PropNetIP6 = "ht:prop:net:ip6";
export const PropNetLatency = "ht:prop:net:latency";
export const PropNetMAC = "ht:prop:net:mac";
export const PropNetMask = "ht:prop:net:mask";
export const PropNetPort = "ht:prop:net:port";
export const PropNetSignalstrength = "ht:prop:net:signalstrength";
export const PropNetSubnet = "ht:prop:net:subnet";
export const PropStatusOnOff = "ht:prop:status:onoff";
export const PropStatusOpenClosed = "ht:prop:status:openclosed";
export const PropStatusStartedStopped = "ht:prop:status:started-stopped";
export const PropStatusYesNo = "ht:prop:status:yes-no";
export const PropSwitch = "ht:prop:switch";
export const PropSwitchDimmer = "ht:prop:switch:dimmer";
export const PropSwitchLight = "ht:prop:switch:light";
export const PropSwitchLocked = "ht:prop:switch:locked";
export const PropSwitchOnOff = "ht:prop:switch:onoff";
// end of PropertyClasses

// PropertyClassesMap maps @type to symbol, title and description
export const PropertyClassesMap = {
  "ht:prop:net:domainname": {Symbol: "", Title: "Domain name", Description: "Domainname of the client"},
  "ht:prop:net:gateway": {Symbol: "", Title: "Gateway", Description: "Network gateway address"},
  "ht:prop:switch:locked": {Symbol: "", Title: "Lock", Description: "Electric lock status"},
  "ht:prop:env:timezone": {Symbol: "", Title: "Timezone", Description: ""},
  "ht:prop:env:vibration": {Symbol: "", Title: "Vibration", Description: ""},
  "ht:prop:location": {Symbol: "", Title: "Location", Description: "General location information"},
  "ht:prop:env:water:level": {Symbol: "", Title: "Water level", Description: ""},
  "ht:prop:location:city": {Symbol: "", Title: "City", Description: "City name"},
  "ht:prop:location:latitude": {Symbol: "", Title: "Latitude", Description: "Latitude geographic coordinate"},
  "ht:prop:location:longitude": {Symbol: "", Title: "Longitude", Description: "Longitude geographic coordinate"},
  "ht:prop:net:latency": {Symbol: "", Title: "Network latency", Description: "Delay between hub and client"},
  "ht:prop:device": {Symbol: "", Title: "Device attributes", Description: "Attributes describing a device"},
  "ht:prop:device:description": {Symbol: "", Title: "Description", Description: "Device product description"},
  "ht:prop:device:enabled-disabled": {Symbol: "", Title: "Enabled/Disabled", Description: "Enabled or disabled state"},
  "ht:prop:device:battery": {Symbol: "", Title: "Battery level", Description: "Device battery level"},
  "ht:prop:env:luminance": {Symbol: "", Title: "Luminance", Description: ""},
  "ht:prop:net:hostname": {Symbol: "", Title: "Hostname", Description: "Hostname of the client"},
  "ht:prop:env": {Symbol: "", Title: "Environmental property", Description: "Property of environmental sensor"},
  "ht:prop:media:track": {Symbol: "", Title: "Track", Description: "Selected A/V track"},
  "ht:prop:net:address": {Symbol: "", Title: "Address", Description: "Network address"},
  "ht:prop:env:barometer": {Symbol: "", Title: "Atmospheric pressure", Description: "Barometric pressure of the atmosphere"},
  "ht:prop:env:wind:speed": {Symbol: "", Title: "Wind speed", Description: ""},
  "ht:prop:switch:onoff": {Symbol: "", Title: "On/Off switch", Description: ""},
  "ht:prop:media:paused": {Symbol: "", Title: "Paused", Description: "Media is paused"},
  "ht:prop:net:mask": {Symbol: "", Title: "Netmask", Description: "Network mask. Example: 255.255.255.0 or 24/8"},
  "ht:prop:device:title": {Symbol: "", Title: "Title", Description: "Device friendly title"},
  "ht:prop:net:mac": {Symbol: "", Title: "MAC", Description: "Hardware MAC address"},
  "ht:prop:net": {Symbol: "", Title: "Network properties", Description: "General network properties"},
  "ht:prop:net:ip6": {Symbol: "", Title: "IP6 address", Description: "Device IP6 address"},
  "ht:prop:net:port": {Symbol: "", Title: "Port", Description: "Network port"},
  "ht:prop:alarm:status": {Symbol: "", Title: "Alarm state", Description: "Current alarm status"},
  "ht:prop:location:street": {Symbol: "", Title: "Street", Description: "Street address"},
  "ht:prop:media:playing": {Symbol: "", Title: "Playing", Description: "Media is playing"},
  "ht:prop:env:humidex": {Symbol: "", Title: "Humidex", Description: ""},
  "ht:prop:env:volume": {Symbol: "", Title: "Volume", Description: ""},
  "ht:prop:location:name": {Symbol: "", Title: "Location name", Description: "Name of the location"},
  "ht:prop:status:openclosed": {Symbol: "", Title: "Open/Closed status", Description: ""},
  "ht:prop:switch": {Symbol: "", Title: "Switch status", Description: ""},
  "ht:prop:device:hardwareversion": {Symbol: "", Title: "Hardware version", Description: ""},
  "ht:prop:electric:current": {Symbol: "", Title: "Current", Description: "Electrical current"},
  "ht:prop:env:co": {Symbol: "", Title: "Carbon monoxide level", Description: "Carbon monoxide level"},
  "ht:prop:media": {Symbol: "", Title: "Media commands", Description: "Control of media equipment"},
  "ht:prop:device:make": {Symbol: "", Title: "Make", Description: "Device manufacturer"},
  "ht:prop:device:softwareversion": {Symbol: "", Title: "Software version", Description: ""},
  "ht:prop:env:wind:heading": {Symbol: "", Title: "Wind heading", Description: ""},
  "ht:prop:device:pollinterval": {Symbol: "", Title: "Polling interval", Description: "Interval to poll for updates"},
  "ht:prop:electric:overload": {Symbol: "", Title: "Overload protection", Description: "Cut load on overload"},
  "ht:prop:env:temperature": {Symbol: "", Title: "Temperature", Description: ""},
  "ht:prop:media:muted": {Symbol: "", Title: "Muted", Description: "Audio is muted"},
  "ht:prop:net:connection": {Symbol: "", Title: "Connection", Description: "Connection status, connected, connecting, retrying, disconnected,..."},
  "ht:prop:status:onoff": {Symbol: "", Title: "On/off status", Description: ""},
  "ht:prop:alarm:motion": {Symbol: "", Title: "Motion", Description: "Motion detected"},
  "ht:prop:electric:poer": {Symbol: "", Title: "Power", Description: "Electrical power being consumed"},
  "ht:prop:env:co2": {Symbol: "", Title: "Carbon dioxide level", Description: "Carbon dioxide level"},
  "ht:prop:net:ip4": {Symbol: "", Title: "IP4 address", Description: "Device IP4 address"},
  "ht:prop:switch:dimmer": {Symbol: "", Title: "Dimmer value", Description: ""},
  "ht:prop:device:status": {Symbol: "", Title: "Status", Description: "Device status; alive, awake, dead, sleeping"},
  "ht:prop:env:fuel:flowrate": {Symbol: "", Title: "Fuel flow rate", Description: ""},
  "ht:prop:media:volume": {Symbol: "", Title: "Volume", Description: "Media volume setting"},
  "ht:prop:env:uv": {Symbol: "", Title: "UV", Description: ""},
  "ht:prop:env:water:flowrate": {Symbol: "", Title: "Water flow rate", Description: ""},
  "ht:prop:net:subnet": {Symbol: "", Title: "Subnet", Description: "Network subnet address. Example: 192.168.0.0"},
  "ht:prop:status:started-stopped": {Symbol: "", Title: "Started/Stopped", Description: "Started or stopped status"},
  "ht:prop:device:model": {Symbol: "", Title: "Model", Description: "Device model"},
  "ht:prop:env:acceleration": {Symbol: "", Title: "Acceleration", Description: ""},
  "ht:prop:env:cpuload": {Symbol: "", Title: "CPU load level", Description: "Device CPU load level"},
  "ht:prop:env:dewpoint": {Symbol: "", Title: "Dew point", Description: "Dew point temperature"},
  "ht:prop:env:fuel:level": {Symbol: "", Title: "Fuel level", Description: ""},
  "ht:prop:env:humidity": {Symbol: "", Title: "Humidity", Description: ""},
  "ht:prop:location:zipcode": {Symbol: "", Title: "Zip code", Description: "Location ZIP code"},
  "ht:prop:media:station": {Symbol: "", Title: "Station", Description: "Selected radio station"},
  "ht:prop:device:firmwareversion": {Symbol: "", Title: "Firmware version", Description: ""},
  "ht:prop:electric:voltage": {Symbol: "", Title: "Voltage", Description: "Electrical voltage potential"},
  "ht:prop:env:airquality": {Symbol: "", Title: "Air quality", Description: "Air quality level"},
  "ht:prop:net:signalstrength": {Symbol: "", Title: "Signal strength", Description: "Wireless signal strength"},
  "ht:prop:switch:light": {Symbol: "", Title: "Light switch", Description: ""},
  "ht:prop:status:yes-no": {Symbol: "", Title: "Yes/No", Description: "Status with yes or no value"},
  "ht:prop:electric": {Symbol: "", Title: "Electrical properties", Description: "General group of electrical properties"},
  "ht:prop:electric:energy": {Symbol: "", Title: "Energy", Description: "Electrical energy consumed"},
  "ht:prop:env:pressure": {Symbol: "", Title: "Pressure", Description: ""},
}


// type: ThingClasses
// version: 0.1
// generated: 19 Nov 24 19:05 PST
// source: github.com/hiveot/hub/api/vocab/ht-thing-classes.yaml
// namespace: ht
export const ThingActuator = "ht:thing:actuator";
export const ThingActuatorAlarm = "ht:thing:actuator:alarm";
export const ThingActuatorBeacon = "ht:thing:actuator:beacon";
export const ThingActuatorDimmer = "ht:thing:actuator:dimmer";
export const ThingActuatorLight = "ht:thing:actuator:light";
export const ThingActuatorLock = "ht:thing:actuator:lock";
export const ThingActuatorMotor = "ht:thing:actuator:motor";
export const ThingActuatorOutput = "ht:thing:actuator:output";
export const ThingActuatorRanged = "ht:thing:actuator:ranged";
export const ThingActuatorRelay = "ht:thing:actuator:relay";
export const ThingActuatorSwitch = "ht:thing:actuator:switch";
export const ThingActuatorValve = "ht:thing:actuator:valve";
export const ThingActuatorValveFuel = "ht:thing:actuator:valve:fuel";
export const ThingActuatorValveWater = "ht:thing:actuator:valve:water";
export const ThingAppliance = "ht:thing:appliance";
export const ThingApplianceDishwasher = "ht:thing:appliance:dishwasher";
export const ThingApplianceDryer = "ht:thing:appliance:dryer";
export const ThingApplianceFreezer = "ht:thing:appliance:freezer";
export const ThingApplianceFridge = "ht:thing:appliance:fridge";
export const ThingApplianceWasher = "ht:thing:appliance:washer";
export const ThingComputer = "ht:thing:computer";
export const ThingComputerCellphone = "ht:thing:computer:cellphone";
export const ThingComputerEmbedded = "ht:thing:computer:embedded";
export const ThingComputerMemory = "ht:thing:computer:memory";
export const ThingComputerPC = "ht:thing:computer:pc";
export const ThingComputerPotsPhone = "ht:thing:computer:potsphone";
export const ThingComputerSatPhone = "ht:thing:computer:satphone";
export const ThingComputerTablet = "ht:thing:computer:tablet";
export const ThingComputerVoipPhone = "ht:thing:computer:voipphone";
export const ThingControl = "ht:thing:control";
export const ThingControlClimate = "ht:thing:control:climate";
export const ThingControlDimmer = "ht:thing:control:dimmer";
export const ThingControlIrrigation = "ht:thing:control:irrigation";
export const ThingControlJoystick = "ht:thing:control:joystick";
export const ThingControlKeypad = "ht:thing:control:keypad";
export const ThingControlPool = "ht:thing:control:pool";
export const ThingControlPushbutton = "ht:thing:control:pushbutton";
export const ThingControlSwitch = "ht:thing:control:switch";
export const ThingControlThermostat = "ht:thing:control:thermostat";
export const ThingControlToggle = "ht:thing:control:toggle";
export const ThingDevice = "ht:thing:device";
export const ThingDeviceBatteryMonitor = "ht:thing:device:battery:monitor";
export const ThingDeviceIndicator = "ht:thing:device:indicator";
export const ThingDeviceTime = "ht:thing:device:time";
export const ThingMedia = "ht:thing:media";
export const ThingMediaAmplifier = "ht:thing:media:amplifier";
export const ThingMediaCamera = "ht:thing:media:camera";
export const ThingMediaMicrophone = "ht:thing:media:microphone";
export const ThingMediaPlayer = "ht:thing:media:player";
export const ThingMediaRadio = "ht:thing:media:radio";
export const ThingMediaReceiver = "ht:thing:media:receiver";
export const ThingMediaSpeaker = "ht:thing:media:speaker";
export const ThingMediaTV = "ht:thing:media:tv";
export const ThingMeter = "ht:thing:meter";
export const ThingMeterElectric = "ht:thing:meter:electric";
export const ThingMeterElectricCurrent = "ht:thing:meter:electric:current";
export const ThingMeterElectricEnergy = "ht:thing:meter:electric:energy";
export const ThingMeterElectricPower = "ht:thing:meter:electric:power";
export const ThingMeterElectricVoltage = "ht:thing:meter:electric:voltage";
export const ThingMeterFuel = "ht:thing:meter:fuel";
export const ThingMeterFuelFlow = "ht:thing:meter:fuel:flow";
export const ThingMeterFuelLevel = "ht:thing:meter:fuel:level";
export const ThingMeterWater = "ht:thing:meter:water";
export const ThingMeterWaterConsumption = "ht:thing:meter:water:consumption";
export const ThingMeterWaterFlow = "ht:thing:meter:water:flow";
export const ThingMeterWaterLevel = "ht:thing:meter:water:level";
export const ThingMeterWind = "ht:thing:meter:wind";
export const ThingNet = "ht:thing:net";
export const ThingNetBluetooth = "ht:thing:net:bluetooth";
export const ThingNetGateway = "ht:thing:net:gateway";
export const ThingNetGatewayCoap = "ht:thing:net:gateway:coap";
export const ThingNetGatewayInsteon = "ht:thing:net:gateway:insteon";
export const ThingNetGatewayOnewire = "ht:thing:net:gateway:onewire";
export const ThingNetGatewayZigbee = "ht:thing:net:gateway:zigbee";
export const ThingNetGatewayZwave = "ht:thing:net:gateway:zwave";
export const ThingNetLora = "ht:thing:net:lora";
export const ThingNetLoraGateway = "ht:thing:net:lora:gw";
export const ThingNetLoraP2P = "ht:thing:net:lora:p2p";
export const ThingNetRouter = "ht:thing:net:router";
export const ThingNetSwitch = "ht:thing:net:switch";
export const ThingNetWifi = "ht:thing:net:wifi";
export const ThingNetWifiAp = "ht:thing:net:wifi:ap";
export const ThingSensor = "ht:thing:sensor";
export const ThingSensorEnvironment = "ht:thing:sensor:environment";
export const ThingSensorInput = "ht:thing:sensor:input";
export const ThingSensorMulti = "ht:thing:sensor:multi";
export const ThingSensorScale = "ht:thing:sensor:scale";
export const ThingSensorSecurity = "ht:thing:sensor:security";
export const ThingSensorSecurityDoorWindow = "ht:thing:sensor:security:doorwindow";
export const ThingSensorSecurityGlass = "ht:thing:sensor:security:glass";
export const ThingSensorSecurityMotion = "ht:thing:sensor:security:motion";
export const ThingSensorSmoke = "ht:thing:sensor:smoke";
export const ThingSensorSound = "ht:thing:sensor:sound";
export const ThingSensorThermometer = "ht:thing:sensor:thermometer";
export const ThingSensorWaterLeak = "ht:thing:sensor:water:leak";
export const ThingService = "ht:thing:service";
// end of ThingClasses

// ThingClassesMap maps @type to symbol, title and description
export const ThingClassesMap = {
  "ht:thing:control:joystick": {Symbol: "", Title: "Joystick", Description: "Flight control stick"},
  "ht:thing:control:toggle": {Symbol: "", Title: "Toggle switch", Description: "Toggle switch input control"},
  "ht:thing:sensor": {Symbol: "", Title: "Sensor", Description: "Generic sensor device"},
  "ht:thing:sensor:security:glass": {Symbol: "", Title: "Glass sensor", Description: "Dedicated sensor for detecting breaking of glass"},
  "ht:thing:computer:cellphone": {Symbol: "", Title: "Cell Phone", Description: "Cellular phone"},
  "ht:thing:computer:voipphone": {Symbol: "", Title: "VoIP Phone", Description: "Voice over IP phone"},
  "ht:thing:appliance:washer": {Symbol: "", Title: "Washer", Description: "Clothing washer"},
  "ht:thing:computer:embedded": {Symbol: "", Title: "Embedded System", Description: "Embedded computing device"},
  "ht:thing:computer:memory": {Symbol: "", Title: "Memory", Description: "Stand-alone memory device such as eeprom or iButtons"},
  "ht:thing:computer:satphone": {Symbol: "", Title: "Satellite phone", Description: ""},
  "ht:thing:control:switch": {Symbol: "", Title: "Input switch", Description: "On or off switch input control"},
  "ht:thing:meter": {Symbol: "", Title: "Meter", Description: "General metering device"},
  "ht:thing:actuator:output": {Symbol: "", Title: "Output", Description: "General purpose electrical output signal"},
  "ht:thing:actuator:relay": {Symbol: "", Title: "Relay", Description: "Generic relay electrical switch"},
  "ht:thing:net:gateway": {Symbol: "", Title: "Gateway", Description: "Generic gateway device providing access to other devices"},
  "ht:thing:service": {Symbol: "", Title: "Service", Description: "General service for processing data and offering features of interest"},
  "ht:thing:actuator:dimmer": {Symbol: "", Title: "Dimmer", Description: "Light dimmer"},
  "ht:thing:sensor:sound": {Symbol: "", Title: "Sound detector", Description: ""},
  "ht:thing:net": {Symbol: "", Title: "Network device", Description: "Generic network device"},
  "ht:thing:actuator:valve": {Symbol: "", Title: "Valve", Description: "Electric powered valve for fluids or gas"},
  "ht:thing:media:amplifier": {Symbol: "", Title: "Audio amplifier", Description: "Audio amplifier with volume controls"},
  "ht:thing:device:battery:monitor": {Symbol: "", Title: "Battery Monitor", Description: "Battery monitor and charge controller"},
  "ht:thing:meter:electric:power": {Symbol: "", Title: "Electrical Power", Description: "Electrical power meter"},
  "ht:thing:appliance:freezer": {Symbol: "", Title: "Freezer", Description: "Refrigerator freezer"},
  "ht:thing:appliance:fridge": {Symbol: "", Title: "Fridge", Description: "Refrigerator appliance"},
  "ht:thing:device": {Symbol: "", Title: "Device", Description: "Device of unknown purpose"},
  "ht:thing:media:microphone": {Symbol: "", Title: "Microphone", Description: "Microphone for capturing audio"},
  "ht:thing:meter:fuel": {Symbol: "", Title: "Fuel metering device", Description: "General fuel metering device"},
  "ht:thing:net:gateway:zigbee": {Symbol: "", Title: "Zigbee gateway", Description: "Gateway providing access to Zigbee devices"},
  "ht:thing:sensor:scale": {Symbol: "", Title: "Scale", Description: "Electronic weigh scale"},
  "ht:thing:actuator:motor": {Symbol: "", Title: "Motor", Description: "Motor driven actuator, such as garage door, blinds, tv lifts"},
  "ht:thing:control:dimmer": {Symbol: "", Title: "Dimmer", Description: "Light dimmer input device"},
  "ht:thing:media:radio": {Symbol: "", Title: "Radio", Description: "AM or FM radio receiver"},
  "ht:thing:meter:wind": {Symbol: "", Title: "Wind", Description: "Dedicated wind meter"},
  "ht:thing:sensor:security:doorwindow": {Symbol: "", Title: "Door/Window sensor", Description: "Dedicated door/window opening security sensor"},
  "ht:thing:sensor:thermometer": {Symbol: "", Title: "Thermometer", Description: "Environmental thermometer"},
  "ht:thing:actuator:light": {Symbol: "", Title: "Light", Description: "Smart LED or other light"},
  "ht:thing:control:pool": {Symbol: "", Title: "Pool control", Description: "Device for controlling pool settings"},
  "ht:thing:control:irrigation": {Symbol: "", Title: "Irrigation control", Description: "Device for control of an irrigation system"},
  "ht:thing:control:thermostat": {Symbol: "", Title: "Thermostat", Description: "Thermostat HVAC control"},
  "ht:thing:media": {Symbol: "", Title: "A/V media", Description: "Generic device for audio/video media record or playback"},
  "ht:thing:net:gateway:insteon": {Symbol: "", Title: "Insteon gateway", Description: "Gateway providing access to Insteon devices"},
  "ht:thing:net:gateway:onewire": {Symbol: "", Title: "1-Wire gateway", Description: "Gateway providing access to 1-wire devices"},
  "ht:thing:computer:pc": {Symbol: "", Title: "PC/Laptop", Description: "Personal computer/laptop"},
  "ht:thing:control": {Symbol: "", Title: "Input controller", Description: "Generic input controller"},
  "ht:thing:actuator:valve:fuel": {Symbol: "", Title: "Fuel valve", Description: "Electric powered fuel valve"},
  "ht:thing:computer:potsphone": {Symbol: "", Title: "Land Line", Description: "Plain Old Telephone System, aka landline"},
  "ht:thing:control:climate": {Symbol: "", Title: "Climate control", Description: "Device for controlling climate of a space"},
  "ht:thing:net:lora:p2p": {Symbol: "", Title: "LoRa P2P", Description: "LoRa Peer-to-peer network device"},
  "ht:thing:sensor:input": {Symbol: "", Title: "Input sensor", Description: "General purpose electrical input sensor"},
  "ht:thing:sensor:security": {Symbol: "", Title: "Security", Description: "Generic security sensor"},
  "ht:thing:actuator:alarm": {Symbol: "", Title: "Alarm", Description: "Siren or light alarm"},
  "ht:thing:actuator:ranged": {Symbol: "", Title: "Ranged actuator", Description: "Generic ranged actuator with a set point"},
  "ht:thing:appliance:dryer": {Symbol: "", Title: "Dryer", Description: "Clothing dryer"},
  "ht:thing:computer": {Symbol: "", Title: "Computing Device", Description: "General purpose computing device"},
  "ht:thing:computer:tablet": {Symbol: "", Title: "Tablet", Description: "Tablet computer"},
  "ht:thing:device:indicator": {Symbol: "", Title: "Indicator", Description: "Visual or audio indicator device"},
  "ht:thing:media:receiver": {Symbol: "", Title: "Receiver", Description: "Audio/video receiver and player"},
  "ht:thing:meter:electric:current": {Symbol: "", Title: "Electric current", Description: "Electrical current meter"},
  "ht:thing:actuator:beacon": {Symbol: "", Title: "Beacon", Description: "Location beacon"},
  "ht:thing:actuator:switch": {Symbol: "", Title: "Switch", Description: "An electric powered on/off switch for powering circuits"},
  "ht:thing:net:lora": {Symbol: "", Title: "LoRa network device", Description: "Generic Long Range network protocol device"},
  "ht:thing:sensor:water:leak": {Symbol: "", Title: "Water leak detector", Description: "Dedicated water leak detector"},
  "ht:thing:meter:water": {Symbol: "", Title: "Water metering device", Description: "General water metering device"},
  "ht:thing:net:switch": {Symbol: "", Title: "Network switch", Description: "Network switch to connect computer devices to the network"},
  "ht:thing:net:bluetooth": {Symbol: "", Title: "Bluetooth", Description: "Bluetooth radio"},
  "ht:thing:net:gateway:zwave": {Symbol: "", Title: "ZWave gateway", Description: "Gateway providing access to ZWave devices"},
  "ht:thing:net:router": {Symbol: "", Title: "Network router", Description: "IP ThingNetwork router providing access to other IP networks"},
  "ht:thing:sensor:environment": {Symbol: "", Title: "Environmental sensor", Description: "Environmental sensor with one or more features such as temperature, humidity, etc"},
  "ht:thing:appliance:dishwasher": {Symbol: "", Title: "Dishwasher", Description: "Dishwasher"},
  "ht:thing:meter:water:consumption": {Symbol: "", Title: "Water consumption meter", Description: "Water consumption meter"},
  "ht:thing:meter:electric:voltage": {Symbol: "", Title: "Voltage", Description: "Electrical voltage meter"},
  "ht:thing:meter:fuel:flow": {Symbol: "", Title: "Fuel flow rate", Description: "Dedicated fuel flow rate metering device"},
  "ht:thing:control:keypad": {Symbol: "", Title: "Keypad", Description: "Multi-key pad for command input"},
  "ht:thing:media:speaker": {Symbol: "", Title: "Connected speakers", Description: "Network connected speakers"},
  "ht:thing:actuator": {Symbol: "", Title: "Actuator", Description: "Generic actuator"},
  "ht:thing:meter:fuel:level": {Symbol: "", Title: "Fuel level", Description: "Dedicated fuel level metering device"},
  "ht:thing:media:player": {Symbol: "", Title: "Media player", Description: "CD/DVD/Blueray/USB player of recorded media"},
  "ht:thing:media:tv": {Symbol: "", Title: "TV", Description: "Network connected television"},
  "ht:thing:sensor:smoke": {Symbol: "", Title: "Smoke detector", Description: ""},
  "ht:thing:actuator:valve:water": {Symbol: "", Title: "Water valve", Description: "Electric powered water valve"},
  "ht:thing:media:camera": {Symbol: "", Title: "Camera", Description: "Video camera"},
  "ht:thing:meter:electric": {Symbol: "", Title: "", Description: ""},
  "ht:thing:meter:water:flow": {Symbol: "", Title: "Water flow", Description: "Dedicated water flow-rate meter"},
  "ht:thing:net:wifi": {Symbol: "", Title: "Wifi device", Description: "Generic wifi device"},
  "ht:thing:net:wifi:ap": {Symbol: "", Title: "Wifi access point", Description: "Wireless access point for IP networks"},
  "ht:thing:net:lora:gw": {Symbol: "", Title: "LoRaWAN gateway", Description: "Gateway providing access to LoRa devices"},
  "ht:thing:control:pushbutton": {Symbol: "", Title: "Momentary switch", Description: "Momentary push button control input"},
  "ht:thing:device:time": {Symbol: "", Title: "Clock", Description: "Time tracking device such as clocks and time chips"},
  "ht:thing:meter:electric:energy": {Symbol: "", Title: "Electric energy", Description: "Electrical energy meter"},
  "ht:thing:meter:water:level": {Symbol: "", Title: "Water level", Description: "Dedicated water level meter"},
  "ht:thing:net:gateway:coap": {Symbol: "", Title: "CoAP gateway", Description: "Gateway providing access to CoAP devices"},
  "ht:thing:sensor:multi": {Symbol: "", Title: "Multi sensor", Description: "Sense multiple inputs"},
  "ht:thing:sensor:security:motion": {Symbol: "", Title: "Motion sensor", Description: "Dedicated security sensor detecting motion"},
  "ht:thing:actuator:lock": {Symbol: "", Title: "Lock", Description: "Electronic door lock"},
  "ht:thing:appliance": {Symbol: "", Title: "Appliance", Description: "Appliance to accomplish a particular task for occupant use"},
}
