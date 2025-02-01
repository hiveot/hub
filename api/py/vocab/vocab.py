# Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions
# DO NOT EDIT. This file is generated and changes will be overwritten

# type: WoTVocab
# version: 0.1
# generated: 31 Jan 25 22:08 PST
# source: github.com/hiveot/hub/api/vocab/wot-vocab.yaml
# description: WoT vocabulary definition. See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
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
# end of WoTVocab

# type: ActionClasses
# version: 0.1
# generated: 31 Jan 25 22:08 PST
# source: github.com/hiveot/hub/api/vocab/ht-action-classes.yaml
# namespace: hiveot
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
# end of ActionClasses

# ActionClassesMap maps @type to symbol, title and description
ActionClassesMap = {
  "hiveot:action:switch:onoff": {"Symbol": "", "Title": "Set On/Off switch", "Description": "Action to set the switch on/off state"},
  "hiveot:action:switch:toggle": {"Symbol": "", "Title": "Toggle switch", "Description": "Action to toggle the switch"},
  "hiveot:action:media": {"Symbol": "", "Title": "Media control", "Description": "Commands to control media recording and playback"},
  "hiveot:action:media:mute": {"Symbol": "", "Title": "Mute", "Description": "Mute audio"},
  "hiveot:action:media:play": {"Symbol": "", "Title": "Play", "Description": "Start or continue playback"},
  "hiveot:action:media:volume:decrease": {"Symbol": "", "Title": "Decrease volume", "Description": "Decrease volume"},
  "hiveot:action:dimmer:decrement": {"Symbol": "", "Title": "Lower dimmer", "Description": ""},
  "hiveot:action:switch": {"Symbol": "", "Title": "Switch", "Description": "General switch action"},
  "hiveot:action:thing:disable": {"Symbol": "", "Title": "Disable", "Description": "Action to disable a thing"},
  "hiveot:action:thing:enable": {"Symbol": "", "Title": "Enable", "Description": "Action to enable a thing"},
  "hiveot:action:media:next": {"Symbol": "", "Title": "Next", "Description": "Next track or station"},
  "hiveot:action:media:volume": {"Symbol": "", "Title": "Volume", "Description": "Set volume level"},
  "hiveot:action:dimmer:increment": {"Symbol": "", "Title": "Increase dimmer", "Description": ""},
  "hiveot:action:valve:open": {"Symbol": "", "Title": "Open valve", "Description": "Action to open the valve"},
  "hiveot:action:valve:close": {"Symbol": "", "Title": "Close valve", "Description": "Action to close the valve"},
  "hiveot:action:media:pause": {"Symbol": "", "Title": "Pause", "Description": "Pause playback"},
  "hiveot:action:media:previous": {"Symbol": "", "Title": "Previous", "Description": "Previous track or station"},
  "hiveot:action:media:unmute": {"Symbol": "", "Title": "Unmute", "Description": "Unmute audio"},
  "hiveot:action:dimmer": {"Symbol": "", "Title": "Dimmer", "Description": "General dimmer action"},
  "hiveot:action:dimmer:set": {"Symbol": "", "Title": "Set dimmer", "Description": "Action to set the dimmer value"},
  "hiveot:action:thing:stop": {"Symbol": "", "Title": "Stop", "Description": "Stop a running task"},
  "hiveot:action:media:volume:increase": {"Symbol": "", "Title": "Increase volume", "Description": "Increase volume"},
  "hiveot:action:thing:start": {"Symbol": "", "Title": "Start", "Description": "Start running a task"},
}


# type: PropertyClasses
# version: 0.1
# generated: 31 Jan 25 22:08 PST
# source: github.com/hiveot/hub/api/vocab/ht-property-classes.yaml
# namespace: hiveot
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
# end of PropertyClasses

# PropertyClassesMap maps @type to symbol, title and description
PropertyClassesMap = {
  "hiveot:prop:device:model": {"Symbol": "", "Title": "Model", "Description": "Device model"},
  "hiveot:prop:env": {"Symbol": "", "Title": "Environmental property", "Description": "Property of environmental sensor"},
  "hiveot:prop:env:cpuload": {"Symbol": "", "Title": "CPU load level", "Description": "Device CPU load level"},
  "hiveot:prop:location:name": {"Symbol": "", "Title": "Location name", "Description": "Name of the location"},
  "hiveot:prop:net:latency": {"Symbol": "", "Title": "Network latency", "Description": "Delay between hub and client"},
  "hiveot:prop:net:subnet": {"Symbol": "", "Title": "Subnet", "Description": "Network subnet address. Example: 192.168.0.0"},
  "hiveot:prop:device": {"Symbol": "", "Title": "Device attributes", "Description": "Attributes describing a device"},
  "hiveot:prop:location:longitude": {"Symbol": "", "Title": "Longitude", "Description": "Longitude geographic coordinate"},
  "hiveot:prop:media:playing": {"Symbol": "", "Title": "Playing", "Description": "Media is playing"},
  "hiveot:prop:net:hostname": {"Symbol": "", "Title": "Hostname", "Description": "Hostname of the client"},
  "hiveot:prop:status:yes-no": {"Symbol": "", "Title": "Yes/No", "Description": "Status with yes or no value"},
  "hiveot:prop:electric:voltage": {"Symbol": "", "Title": "Voltage", "Description": "Electrical voltage potential"},
  "hiveot:prop:env:barometer": {"Symbol": "", "Title": "Atmospheric pressure", "Description": "Barometric pressure of the atmosphere"},
  "hiveot:prop:env:fuel:flowrate": {"Symbol": "", "Title": "Fuel flow rate", "Description": ""},
  "hiveot:prop:env:uv": {"Symbol": "", "Title": "UV", "Description": ""},
  "hiveot:prop:device:softwareversion": {"Symbol": "", "Title": "Software version", "Description": ""},
  "hiveot:prop:env:pressure": {"Symbol": "", "Title": "Pressure", "Description": ""},
  "hiveot:prop:env:water:flowrate": {"Symbol": "", "Title": "Water flow rate", "Description": ""},
  "hiveot:prop:media": {"Symbol": "", "Title": "Media commands", "Description": "Control of media equipment"},
  "hiveot:prop:alarm:motion": {"Symbol": "", "Title": "Motion", "Description": "Motion detected"},
  "hiveot:prop:electric": {"Symbol": "", "Title": "Electrical properties", "Description": "General group of electrical properties"},
  "hiveot:prop:env:co2": {"Symbol": "", "Title": "Carbon dioxide level", "Description": "Carbon dioxide level"},
  "hiveot:prop:electric:poer": {"Symbol": "", "Title": "Power", "Description": "Electrical power being consumed"},
  "hiveot:prop:env:airquality": {"Symbol": "", "Title": "Air quality", "Description": "Air quality level"},
  "hiveot:prop:env:humidity": {"Symbol": "", "Title": "Humidity", "Description": ""},
  "hiveot:prop:env:timezone": {"Symbol": "", "Title": "Timezone", "Description": ""},
  "hiveot:prop:env:wind:heading": {"Symbol": "", "Title": "Wind heading", "Description": ""},
  "hiveot:prop:status:onoff": {"Symbol": "", "Title": "On/off status", "Description": ""},
  "hiveot:prop:electric:current": {"Symbol": "", "Title": "Current", "Description": "Electrical current"},
  "hiveot:prop:env:dewpoint": {"Symbol": "", "Title": "Dew point", "Description": "Dew point temperature"},
  "hiveot:prop:env:water:level": {"Symbol": "", "Title": "Water level", "Description": ""},
  "hiveot:prop:net:gateway": {"Symbol": "", "Title": "Gateway", "Description": "Network gateway address"},
  "hiveot:prop:net:ip4": {"Symbol": "", "Title": "IP4 address", "Description": "Device IP4 address"},
  "hiveot:prop:net:ip6": {"Symbol": "", "Title": "IP6 address", "Description": "Device IP6 address"},
  "hiveot:prop:env:acceleration": {"Symbol": "", "Title": "Acceleration", "Description": ""},
  "hiveot:prop:env:humidex": {"Symbol": "", "Title": "Humidex", "Description": ""},
  "hiveot:prop:env:wind:speed": {"Symbol": "", "Title": "Wind speed", "Description": ""},
  "hiveot:prop:media:muted": {"Symbol": "", "Title": "Muted", "Description": "Audio is muted"},
  "hiveot:prop:net:port": {"Symbol": "", "Title": "Port", "Description": "Network port"},
  "hiveot:prop:switch": {"Symbol": "", "Title": "Switch status", "Description": ""},
  "hiveot:prop:switch:onoff": {"Symbol": "", "Title": "On/Off switch", "Description": ""},
  "hiveot:prop:alarm:status": {"Symbol": "", "Title": "Alarm state", "Description": "Current alarm status"},
  "hiveot:prop:device:description": {"Symbol": "", "Title": "Description", "Description": "Device product description"},
  "hiveot:prop:media:track": {"Symbol": "", "Title": "Track", "Description": "Selected A/V track"},
  "hiveot:prop:net:mask": {"Symbol": "", "Title": "Netmask", "Description": "Network mask. Example: 255.255.255.0 or 24/8"},
  "hiveot:prop:net:signalstrength": {"Symbol": "", "Title": "Signal strength", "Description": "Wireless signal strength"},
  "hiveot:prop:status:started-stopped": {"Symbol": "", "Title": "Started/Stopped", "Description": "Started or stopped status"},
  "hiveot:prop:device:make": {"Symbol": "", "Title": "Make", "Description": "Device manufacturer"},
  "hiveot:prop:location:latitude": {"Symbol": "", "Title": "Latitude", "Description": "Latitude geographic coordinate"},
  "hiveot:prop:location:zipcode": {"Symbol": "", "Title": "Zip code", "Description": "Location ZIP code"},
  "hiveot:prop:net:connection": {"Symbol": "", "Title": "Connection", "Description": "Connection status, connected, connecting, retrying, disconnected,..."},
  "hiveot:prop:net:mac": {"Symbol": "", "Title": "MAC", "Description": "Hardware MAC address"},
  "hiveot:prop:status:openclosed": {"Symbol": "", "Title": "Open/Closed status", "Description": ""},
  "hiveot:prop:switch:locked": {"Symbol": "", "Title": "Lock", "Description": "Electric lock status"},
  "hiveot:prop:device:enabled-disabled": {"Symbol": "", "Title": "Enabled/Disabled", "Description": "Enabled or disabled state"},
  "hiveot:prop:electric:overload": {"Symbol": "", "Title": "Overload protection", "Description": "Cut load on overload"},
  "hiveot:prop:env:co": {"Symbol": "", "Title": "Carbon monoxide level", "Description": "Carbon monoxide level"},
  "hiveot:prop:env:fuel:level": {"Symbol": "", "Title": "Fuel level", "Description": ""},
  "hiveot:prop:env:luminance": {"Symbol": "", "Title": "Luminance", "Description": ""},
  "hiveot:prop:net": {"Symbol": "", "Title": "Network properties", "Description": "General network properties"},
  "hiveot:prop:device:pollinterval": {"Symbol": "", "Title": "Polling interval", "Description": "Interval to poll for updates"},
  "hiveot:prop:device:status": {"Symbol": "", "Title": "Status", "Description": "Device status; alive, awake, dead, sleeping"},
  "hiveot:prop:net:domainname": {"Symbol": "", "Title": "Domain name", "Description": "Domainname of the client"},
  "hiveot:prop:device:hardwareversion": {"Symbol": "", "Title": "Hardware version", "Description": ""},
  "hiveot:prop:electric:energy": {"Symbol": "", "Title": "Energy", "Description": "Electrical energy consumed"},
  "hiveot:prop:env:temperature": {"Symbol": "", "Title": "Temperature", "Description": ""},
  "hiveot:prop:location:street": {"Symbol": "", "Title": "Street", "Description": "Street address"},
  "hiveot:prop:media:paused": {"Symbol": "", "Title": "Paused", "Description": "Media is paused"},
  "hiveot:prop:net:address": {"Symbol": "", "Title": "Address", "Description": "Network address"},
  "hiveot:prop:switch:dimmer": {"Symbol": "", "Title": "Dimmer value", "Description": ""},
  "hiveot:prop:switch:light": {"Symbol": "", "Title": "Light switch", "Description": ""},
  "hiveot:prop:device:battery": {"Symbol": "", "Title": "Battery level", "Description": "Device battery level"},
  "hiveot:prop:device:firmwareversion": {"Symbol": "", "Title": "Firmware version", "Description": ""},
  "hiveot:prop:device:title": {"Symbol": "", "Title": "Title", "Description": "Device friendly title"},
  "hiveot:prop:env:vibration": {"Symbol": "", "Title": "Vibration", "Description": ""},
  "hiveot:prop:location:city": {"Symbol": "", "Title": "City", "Description": "City name"},
  "hiveot:prop:media:station": {"Symbol": "", "Title": "Station", "Description": "Selected radio station"},
  "hiveot:prop:env:volume": {"Symbol": "", "Title": "Volume", "Description": ""},
  "hiveot:prop:location": {"Symbol": "", "Title": "Location", "Description": "General location information"},
  "hiveot:prop:media:volume": {"Symbol": "", "Title": "Volume", "Description": "Media volume setting"},
}


# type: ThingClasses
# version: 0.1
# generated: 31 Jan 25 22:08 PST
# source: github.com/hiveot/hub/api/vocab/ht-thing-classes.yaml
# namespace: hiveot
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
# end of ThingClasses

# ThingClassesMap maps @type to symbol, title and description
ThingClassesMap = {
  "hiveot:thing:control:thermostat": {"Symbol": "", "Title": "Thermostat", "Description": "Thermostat HVAC control"},
  "hiveot:thing:media:speaker": {"Symbol": "", "Title": "Connected speakers", "Description": "Network connected speakers"},
  "hiveot:thing:net:gateway:zigbee": {"Symbol": "", "Title": "Zigbee gateway", "Description": "Gateway providing access to Zigbee devices"},
  "hiveot:thing:net:lora": {"Symbol": "", "Title": "LoRa network device", "Description": "Generic Long Range network protocol device"},
  "hiveot:thing:sensor": {"Symbol": "", "Title": "Sensor", "Description": "Generic sensor device"},
  "hiveot:thing:computer:voipphone": {"Symbol": "", "Title": "VoIP Phone", "Description": "Voice over IP phone"},
  "hiveot:thing:control:dimmer": {"Symbol": "", "Title": "Dimmer", "Description": "Light dimmer input device"},
  "hiveot:thing:control:joystick": {"Symbol": "", "Title": "Joystick", "Description": "Flight control stick"},
  "hiveot:thing:sensor:security:glass": {"Symbol": "", "Title": "Glass sensor", "Description": "Dedicated sensor for detecting breaking of glass"},
  "hiveot:thing:sensor:security:doorwindow": {"Symbol": "", "Title": "Door/Window sensor", "Description": "Dedicated door/window opening security sensor"},
  "hiveot:thing:actuator:ranged": {"Symbol": "", "Title": "Ranged actuator", "Description": "Generic ranged actuator with a set point"},
  "hiveot:thing:computer:cellphone": {"Symbol": "", "Title": "Cell Phone", "Description": "Cellular phone"},
  "hiveot:thing:net:gateway": {"Symbol": "", "Title": "Gateway", "Description": "Generic gateway device providing access to other devices"},
  "hiveot:thing:service": {"Symbol": "", "Title": "Service", "Description": "General service for processing data and offering features of interest"},
  "hiveot:thing:computer:embedded": {"Symbol": "", "Title": "Embedded System", "Description": "Embedded computing device"},
  "hiveot:thing:control": {"Symbol": "", "Title": "Input controller", "Description": "Generic input controller"},
  "hiveot:thing:device": {"Symbol": "", "Title": "Device", "Description": "Device of unknown purpose"},
  "hiveot:thing:device:indicator": {"Symbol": "", "Title": "Indicator", "Description": "Visual or audio indicator device"},
  "hiveot:thing:media": {"Symbol": "", "Title": "A/V media", "Description": "Generic device for audio/video media record or playback"},
  "hiveot:thing:actuator:beacon": {"Symbol": "", "Title": "Beacon", "Description": "Location beacon"},
  "hiveot:thing:actuator:output": {"Symbol": "", "Title": "Output", "Description": "General purpose electrical output signal"},
  "hiveot:thing:appliance": {"Symbol": "", "Title": "Appliance", "Description": "Appliance to accomplish a particular task for occupant use"},
  "hiveot:thing:net:gateway:insteon": {"Symbol": "", "Title": "Insteon gateway", "Description": "Gateway providing access to Insteon devices"},
  "hiveot:thing:meter:electric:energy": {"Symbol": "", "Title": "Electric energy", "Description": "Electrical energy meter"},
  "hiveot:thing:meter:fuel:flow": {"Symbol": "", "Title": "Fuel flow rate", "Description": "Dedicated fuel flow rate metering device"},
  "hiveot:thing:meter:fuel:level": {"Symbol": "", "Title": "Fuel level", "Description": "Dedicated fuel level metering device"},
  "hiveot:thing:meter": {"Symbol": "", "Title": "Meter", "Description": "General metering device"},
  "hiveot:thing:net:wifi:ap": {"Symbol": "", "Title": "Wifi access point", "Description": "Wireless access point for IP networks"},
  "hiveot:thing:sensor:input": {"Symbol": "", "Title": "Input sensor", "Description": "General purpose electrical input sensor"},
  "hiveot:thing:actuator:switch": {"Symbol": "", "Title": "Switch", "Description": "An electric powered on/off switch for powering circuits"},
  "hiveot:thing:actuator:valve": {"Symbol": "", "Title": "Valve", "Description": "Electric powered valve for fluids or gas"},
  "hiveot:thing:control:pool": {"Symbol": "", "Title": "Pool control", "Description": "Device for controlling pool settings"},
  "hiveot:thing:media:camera": {"Symbol": "", "Title": "Camera", "Description": "Video camera"},
  "hiveot:thing:media:player": {"Symbol": "", "Title": "Media player", "Description": "CD/DVD/Blueray/USB player of recorded media"},
  "hiveot:thing:net:wifi": {"Symbol": "", "Title": "Wifi device", "Description": "Generic wifi device"},
  "hiveot:thing:actuator:relay": {"Symbol": "", "Title": "Relay", "Description": "Generic relay electrical switch"},
  "hiveot:thing:actuator:valve:water": {"Symbol": "", "Title": "Water valve", "Description": "Electric powered water valve"},
  "hiveot:thing:computer": {"Symbol": "", "Title": "Computing Device", "Description": "General purpose computing device"},
  "hiveot:thing:net:bluetooth": {"Symbol": "", "Title": "Bluetooth", "Description": "Bluetooth radio"},
  "hiveot:thing:net:gateway:zwave": {"Symbol": "", "Title": "ZWave gateway", "Description": "Gateway providing access to ZWave devices"},
  "hiveot:thing:sensor:security": {"Symbol": "", "Title": "Security", "Description": "Generic security sensor"},
  "hiveot:thing:control:pushbutton": {"Symbol": "", "Title": "Momentary switch", "Description": "Momentary push button control input"},
  "hiveot:thing:device:battery:monitor": {"Symbol": "", "Title": "Battery Monitor", "Description": "Battery monitor and charge controller"},
  "hiveot:thing:media:tv": {"Symbol": "", "Title": "TV", "Description": "Network connected television"},
  "hiveot:thing:appliance:freezer": {"Symbol": "", "Title": "Freezer", "Description": "Refrigerator freezer"},
  "hiveot:thing:meter:water:level": {"Symbol": "", "Title": "Water level", "Description": "Dedicated water level meter"},
  "hiveot:thing:actuator:motor": {"Symbol": "", "Title": "Motor", "Description": "Motor driven actuator, such as garage door, blinds, tv lifts"},
  "hiveot:thing:computer:tablet": {"Symbol": "", "Title": "Tablet", "Description": "Tablet computer"},
  "hiveot:thing:net:router": {"Symbol": "", "Title": "Network router", "Description": "IP ThingNetwork router providing access to other IP networks"},
  "hiveot:thing:net:lora:p2p": {"Symbol": "", "Title": "LoRa P2P", "Description": "LoRa Peer-to-peer network device"},
  "hiveot:thing:actuator:dimmer": {"Symbol": "", "Title": "Dimmer", "Description": "Light dimmer"},
  "hiveot:thing:actuator:lock": {"Symbol": "", "Title": "Lock", "Description": "Electronic door lock"},
  "hiveot:thing:actuator:light": {"Symbol": "", "Title": "Light", "Description": "Smart LED or other light"},
  "hiveot:thing:sensor:smoke": {"Symbol": "", "Title": "Smoke detector", "Description": ""},
  "hiveot:thing:appliance:dishwasher": {"Symbol": "", "Title": "Dishwasher", "Description": "Dishwasher"},
  "hiveot:thing:meter:fuel": {"Symbol": "", "Title": "Fuel metering device", "Description": "General fuel metering device"},
  "hiveot:thing:sensor:environment": {"Symbol": "", "Title": "Environmental sensor", "Description": "Environmental sensor with one or more features such as temperature, humidity, etc"},
  "hiveot:thing:net:gateway:coap": {"Symbol": "", "Title": "CoAP gateway", "Description": "Gateway providing access to CoAP devices"},
  "hiveot:thing:net:gateway:onewire": {"Symbol": "", "Title": "1-Wire gateway", "Description": "Gateway providing access to 1-wire devices"},
  "hiveot:thing:sensor:multi": {"Symbol": "", "Title": "Multi sensor", "Description": "Sense multiple inputs"},
  "hiveot:thing:device:time": {"Symbol": "", "Title": "Clock", "Description": "Time tracking device such as clocks and time chips"},
  "hiveot:thing:media:receiver": {"Symbol": "", "Title": "Receiver", "Description": "Audio/video receiver and player"},
  "hiveot:thing:meter:water:consumption": {"Symbol": "", "Title": "Water consumption meter", "Description": "Water consumption meter"},
  "hiveot:thing:control:toggle": {"Symbol": "", "Title": "Toggle switch", "Description": "Toggle switch input control"},
  "hiveot:thing:meter:electric:voltage": {"Symbol": "", "Title": "Voltage", "Description": "Electrical voltage meter"},
  "hiveot:thing:net:lora:gw": {"Symbol": "", "Title": "LoRaWAN gateway", "Description": "Gateway providing access to LoRa devices"},
  "hiveot:thing:net:switch": {"Symbol": "", "Title": "Network switch", "Description": "Network switch to connect computer devices to the network"},
  "hiveot:thing:sensor:security:motion": {"Symbol": "", "Title": "Motion sensor", "Description": "Dedicated security sensor detecting motion"},
  "hiveot:thing:sensor:sound": {"Symbol": "", "Title": "Sound detector", "Description": ""},
  "hiveot:thing:actuator:alarm": {"Symbol": "", "Title": "Alarm", "Description": "Siren or light alarm"},
  "hiveot:thing:computer:pc": {"Symbol": "", "Title": "PC/Laptop", "Description": "Personal computer/laptop"},
  "hiveot:thing:media:radio": {"Symbol": "", "Title": "Radio", "Description": "AM or FM radio receiver"},
  "hiveot:thing:media:microphone": {"Symbol": "", "Title": "Microphone", "Description": "Microphone for capturing audio"},
  "hiveot:thing:meter:electric": {"Symbol": "", "Title": "", "Description": ""},
  "hiveot:thing:meter:electric:current": {"Symbol": "", "Title": "Electric current", "Description": "Electrical current meter"},
  "hiveot:thing:meter:electric:power": {"Symbol": "", "Title": "Electrical Power", "Description": "Electrical power meter"},
  "hiveot:thing:sensor:thermometer": {"Symbol": "", "Title": "Thermometer", "Description": "Environmental thermometer"},
  "hiveot:thing:appliance:dryer": {"Symbol": "", "Title": "Dryer", "Description": "Clothing dryer"},
  "hiveot:thing:appliance:fridge": {"Symbol": "", "Title": "Fridge", "Description": "Refrigerator appliance"},
  "hiveot:thing:computer:satphone": {"Symbol": "", "Title": "Satellite phone", "Description": ""},
  "hiveot:thing:sensor:water:leak": {"Symbol": "", "Title": "Water leak detector", "Description": "Dedicated water leak detector"},
  "hiveot:thing:control:switch": {"Symbol": "", "Title": "Input switch", "Description": "On or off switch input control"},
  "hiveot:thing:media:amplifier": {"Symbol": "", "Title": "Audio amplifier", "Description": "Audio amplifier with volume controls"},
  "hiveot:thing:meter:wind": {"Symbol": "", "Title": "Wind", "Description": "Dedicated wind meter"},
  "hiveot:thing:actuator:valve:fuel": {"Symbol": "", "Title": "Fuel valve", "Description": "Electric powered fuel valve"},
  "hiveot:thing:computer:memory": {"Symbol": "", "Title": "Memory", "Description": "Stand-alone memory device such as eeprom or iButtons"},
  "hiveot:thing:control:keypad": {"Symbol": "", "Title": "Keypad", "Description": "Multi-key pad for command input"},
  "hiveot:thing:control:climate": {"Symbol": "", "Title": "Climate control", "Description": "Device for controlling climate of a space"},
  "hiveot:thing:control:irrigation": {"Symbol": "", "Title": "Irrigation control", "Description": "Device for control of an irrigation system"},
  "hiveot:thing:meter:water": {"Symbol": "", "Title": "Water metering device", "Description": "General water metering device"},
  "hiveot:thing:meter:water:flow": {"Symbol": "", "Title": "Water flow", "Description": "Dedicated water flow-rate meter"},
  "hiveot:thing:net": {"Symbol": "", "Title": "Network device", "Description": "Generic network device"},
  "hiveot:thing:actuator": {"Symbol": "", "Title": "Actuator", "Description": "Generic actuator"},
  "hiveot:thing:appliance:washer": {"Symbol": "", "Title": "Washer", "Description": "Clothing washer"},
  "hiveot:thing:computer:potsphone": {"Symbol": "", "Title": "Land Line", "Description": "Plain Old Telephone System, aka landline"},
  "hiveot:thing:sensor:scale": {"Symbol": "", "Title": "Scale", "Description": "Electronic weigh scale"},
}


# type: UnitClasses
# version: 0.1
# generated: 31 Jan 25 22:08 PST
# source: github.com/hiveot/hub/api/vocab/ht-unit-classes.yaml
# namespace: hiveot
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
# end of UnitClasses

# UnitClassesMap maps @type to symbol, title and description
UnitClassesMap = {
  "hiveot:unit:watt": {"Symbol": "W", "Title": "Watt", "Description": "SI unit of power. Equal to 1 joule per second; or work performed when a current of 1 ampere flows across an electric potential of one volt."},
  "hiveot:unit:ppm": {"Symbol": "ppm", "Title": "PPM", "Description": "Parts per million"},
  "hiveot:unit:fahrenheit": {"Symbol": "F", "Title": "Fahrenheit", "Description": "Temperature in Fahrenheit"},
  "hiveot:unit:meter": {"Symbol": "m", "Title": "Meter", "Description": "Distance in meters. 1m=c/299792458"},
  "hiveot:unit:meterspersecond": {"Symbol": "m/s", "Title": "Meters per second", "Description": "SI unit of speed in meters per second"},
  "hiveot:unit:millibar": {"Symbol": "mbar", "Title": "millibar", "Description": "Metric unit of pressure. 1/1000th of a bar. Equal to 100 pascals. Amount of force it takes to move an object weighing a gram, one centimeter in one second."},
  "hiveot:unit:second": {"Symbol": "s", "Title": "Second", "Description": "SI unit of time based on caesium frequency"},
  "hiveot:unit:celcius": {"Symbol": "°C", "Title": "Celcius", "Description": "Temperature in Celcius"},
  "hiveot:unit:kilowatthour": {"Symbol": "kWh", "Title": "Kilowatt-hour", "Description": "non-SI unit of energy equivalent to 3.6 megajoules."},
  "hiveot:unit:volt": {"Symbol": "V", "Title": "Volt", "Description": "SI unit of electric potential; Energy consumption of 1 joule per electric charge of one coulomb"},
  "hiveot:unit:candela": {"Symbol": "cd", "Title": "Candela", "Description": "SI unit of luminous intensity in a given direction. Roughly the same brightness as the common candle."},
  "hiveot:unit:lumen": {"Symbol": "lm", "Title": "Lumen", "Description": "SI unit luminous flux. Measure of perceived power of visible light. 1lm = 1 cd steradian"},
  "hiveot:unit:pound": {"Symbol": "lbs", "Title": "Pound", "Description": "Imperial unit of weight. Equivalent to 0.453592 Kg. 1 Kg is 2.205 lbs"},
  "hiveot:unit:liter": {"Symbol": "l", "Title": "Liter", "Description": "SI unit of volume equivalent to 1 cubic decimeter."},
  "hiveot:unit:kph": {"Symbol": "kph", "Title": "Km per hour", "Description": "Speed in kilometers per hour"},
  "hiveot:unit:millisecond": {"Symbol": "ms", "Title": "millisecond", "Description": "Unit of time in milli-seconds. Equal to 1/1000 of a second."},
  "hiveot:unit:radian": {"Symbol": "", "Title": "Radian", "Description": "Angle in 0-2pi"},
  "hiveot:unit:kilogram": {"Symbol": "kg", "Title": "Kilogram", "Description": ""},
  "hiveot:unit:gallon": {"Symbol": "gl", "Title": "Gallon", "Description": "Unit of volume. 1 Imperial gallon is 4.54609 liters. 1 US liquid gallon is 3.78541 liters. 1 US dry gallon is 4.405 liters. "},
  "hiveot:unit:milesperhour": {"Symbol": "mph", "Title": "Miles per hour", "Description": "Speed in miles per hour"},
  "hiveot:unit:pascal": {"Symbol": "Pa", "Title": "Pascal", "Description": "SI unit of pressure. Equal to 1 newton of force applied over 1 square meter."},
  "hiveot:unit:psi": {"Symbol": "PSI", "Title": "PSI", "Description": "Unit of pressure. Pounds of force per square inch. 1PSI equals 6984 Pascals."},
  "hiveot:unit:count": {"Symbol": "(N)", "Title": "Count", "Description": ""},
  "hiveot:unit:foot": {"Symbol": "ft", "Title": "Foot", "Description": "Imperial unit of distance. 1 foot equals 0.3048 meters"},
  "hiveot:unit:kelvin": {"Symbol": "K", "Title": "Kelvin", "Description": "SI unit of thermodynamic temperature. 0 K represents absolute zero, the absence of all heat. 0 C equals +273.15K"},
  "hiveot:unit:lux": {"Symbol": "lx", "Title": "Lux", "Description": "SI unit illuminance. Equal to 1 lumen per square meter."},
  "hiveot:unit:mercury": {"Symbol": "Hg", "Title": "Mercury", "Description": "Unit of atmospheric pressure in the United States. 1 Hg equals 33.8639 mbar."},
  "hiveot:unit:mole": {"Symbol": "mol", "Title": "Mole", "Description": "SI unit of measurement for amount of substance. Eg, molecules."},
  "hiveot:unit:degree": {"Symbol": "degree", "Title": "Degree", "Description": "Angle in 0-360 degrees"},
  "hiveot:unit:percent": {"Symbol": "%", "Title": "Percent", "Description": "Fractions of 100"},
  "hiveot:unit:ampere": {"Symbol": "A", "Title": "Ampere", "Description": "Electrical current in Amperes based on the elementary charge flow per second"},
}
