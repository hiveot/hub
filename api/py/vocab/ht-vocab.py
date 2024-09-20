# Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions
# DO NOT EDIT. This file is generated and changes will be overwritten

# type: MessageTypes
# version: 0.1
# generated: 19 Sep 24 19:02 PDT
# source: github.com/hiveot/hub/api/vocab/ht-constants.yaml
# description: Message types used throughout the Hub and its clients
EventNameDeliveryUpdate = "$delivery"
EventNameProperties = "$properties"
EventNameTD = "$td"
MessageTypeAction = "action"
MessageTypeDeliveryUpdate = "delivery"
MessageTypeEvent = "event"
MessageTypeProperty = "property"
# end of MessageTypes

# type: WoTVocab
# version: 0.1
# generated: 19 Sep 24 19:02 PDT
# source: github.com/hiveot/hub/api/vocab/wot-vocab.yaml
# description: WoT vocabulary definition. See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
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
WoTOpObserveProperty = "observeproperty"
WoTOpReadProperty = "readproperty"
WoTOpUnobserveProperty = "unobserveroperty"
WoTOpWriteProperty = "writeproperty"
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
WotOpCancelAction = "cancelaction"
WotOpInvokeAction = "invokeaction"
WotOpObserveAllProperties = "observeallproperties"
WotOpQueryAction = "queryaction"
WotOpQueryAllAction = "queryallactions"
WotOpReadAllProperties = "readallproperties"
WotOpReadMultipleProperties = "readmultipleproperties"
WotOpSubscribeAllEvents = "subscribeallevents"
WotOpSubscribeEvent = "subscribeevent"
WotOpUnobserveAllProperties = "unobserveallproperties"
WotOpUnsubscribeAllEvents = "unsubscribeallevents"
WotOpUnsubscribeEvent = "unsubscribeevent"
WotOpWriteAllProperties = "writeallproperties"
WotOpWriteMultipleProperties = "writemultipleproperties"
# end of WoTVocab

# type: ActionClasses
# version: 0.1
# generated: 19 Sep 24 19:02 PDT
# source: github.com/hiveot/hub/api/vocab/ht-action-classes.yaml
# namespace: ht
ActionDimmer = "ht:action:dimmer"
ActionDimmerDecrement = "ht:action:dimmer:decrement"
ActionDimmerIncrement = "ht:action:dimmer:increment"
ActionDimmerSet = "ht:action:dimmer:set"
ActionMedia = "ht:action:media"
ActionMediaMute = "ht:action:media:mute"
ActionMediaNext = "ht:action:media:next"
ActionMediaPause = "ht:action:media:pause"
ActionMediaPlay = "ht:action:media:play"
ActionMediaPrevious = "ht:action:media:previous"
ActionMediaUnmute = "ht:action:media:unmute"
ActionMediaVolume = "ht:action:media:volume"
ActionMediaVolumeDecrease = "ht:action:media:volume:decrease"
ActionMediaVolumeIncrease = "ht:action:media:volume:increase"
ActionSwitch = "ht:action:switch"
ActionSwitchOff = "ht:action:switch:off"
ActionSwitchOn = "ht:action:switch:on"
ActionSwitchOnOff = "ht:action:switch:onoff"
ActionSwitchToggle = "ht:action:switch:toggle"
ActionThingDisable = "ht:action:thing:disable"
ActionThingEnable = "ht:action:thing:enable"
ActionThingStart = "ht:action:thing:start"
ActionThingStop = "ht:action:thing:stop"
ActionValveClose = "ht:action:valve:close"
ActionValveOpen = "ht:action:valve:open"
# end of ActionClasses

# ActionClassesMap maps @type to symbol, title and description
ActionClassesMap = {
  "ht:action:media:volume": {"Symbol": "", "Title": "Volume", "Description": "Set volume level"},
  "ht:action:switch:on": {"Symbol": "", "Title": "Switch on", "Description": "Action to turn the switch on"},
  "ht:action:valve:close": {"Symbol": "", "Title": "Close valve", "Description": "Action to close the valve"},
  "ht:action:media:previous": {"Symbol": "", "Title": "Previous", "Description": "Previous track or station"},
  "ht:action:dimmer:increment": {"Symbol": "", "Title": "Increase dimmer", "Description": ""},
  "ht:action:switch": {"Symbol": "", "Title": "Switch", "Description": "General switch action"},
  "ht:action:switch:onoff": {"Symbol": "", "Title": "Set On/Off switch", "Description": "Action to set the switch on/off state"},
  "ht:action:switch:toggle": {"Symbol": "", "Title": "Toggle switch", "Description": "Action to toggle the switch"},
  "ht:action:thing:disable": {"Symbol": "", "Title": "Disable", "Description": "Action to disable a thing"},
  "ht:action:thing:enable": {"Symbol": "", "Title": "Enable", "Description": "Action to enable a thing"},
  "ht:action:valve:open": {"Symbol": "", "Title": "Open valve", "Description": "Action to open the valve"},
  "ht:action:media:mute": {"Symbol": "", "Title": "Mute", "Description": "Mute audio"},
  "ht:action:media:volume:decrease": {"Symbol": "", "Title": "Decrease volume", "Description": "Decrease volume"},
  "ht:action:dimmer": {"Symbol": "", "Title": "Dimmer", "Description": "General dimmer action"},
  "ht:action:thing:stop": {"Symbol": "", "Title": "Stop", "Description": "Stop a running task"},
  "ht:action:media:volume:increase": {"Symbol": "", "Title": "Increase volume", "Description": "Increase volume"},
  "ht:action:media:next": {"Symbol": "", "Title": "Next", "Description": "Next track or station"},
  "ht:action:media:pause": {"Symbol": "", "Title": "Pause", "Description": "Pause playback"},
  "ht:action:media:play": {"Symbol": "", "Title": "Play", "Description": "Start or continue playback"},
  "ht:action:media:unmute": {"Symbol": "", "Title": "Unmute", "Description": "Unmute audio"},
  "ht:action:dimmer:decrement": {"Symbol": "", "Title": "Lower dimmer", "Description": ""},
  "ht:action:dimmer:set": {"Symbol": "", "Title": "Set dimmer", "Description": "Action to set the dimmer value"},
  "ht:action:switch:off": {"Symbol": "", "Title": "Switch off", "Description": "Action to turn the switch off"},
  "ht:action:media": {"Symbol": "", "Title": "Media control", "Description": "Commands to control media recording and playback"},
  "ht:action:thing:start": {"Symbol": "", "Title": "Start", "Description": "Start running a task"},
}


# type: PropertyClasses
# version: 0.1
# generated: 19 Sep 24 19:02 PDT
# source: github.com/hiveot/hub/api/vocab/ht-property-classes.yaml
# namespace: ht
PropAlarmMotion = "ht:prop:alarm:motion"
PropAlarmStatus = "ht:prop:alarm:status"
PropDevice = "ht:prop:device"
PropDeviceBattery = "ht:prop:device:battery"
PropDeviceDescription = "ht:prop:device:description"
PropDeviceEnabledDisabled = "ht:prop:device:enabled-disabled"
PropDeviceFirmwareVersion = "ht:prop:device:firmwareversion"
PropDeviceHardwareVersion = "ht:prop:device:hardwareversion"
PropDeviceMake = "ht:prop:device:make"
PropDeviceModel = "ht:prop:device:model"
PropDevicePollinterval = "ht:prop:device:pollinterval"
PropDeviceSoftwareVersion = "ht:prop:device:softwareversion"
PropDeviceStatus = "ht:prop:device:status"
PropDeviceTitle = "ht:prop:device:title"
PropElectric = "ht:prop:electric"
PropElectricCurrent = "ht:prop:electric:current"
PropElectricEnergy = "ht:prop:electric:energy"
PropElectricOverload = "ht:prop:electric:overload"
PropElectricPower = "ht:prop:electric:poer"
PropElectricVoltage = "ht:prop:electric:voltage"
PropEnv = "ht:prop:env"
PropEnvAcceleration = "ht:prop:env:acceleration"
PropEnvAirquality = "ht:prop:env:airquality"
PropEnvBarometer = "ht:prop:env:barometer"
PropEnvCO = "ht:prop:env:co"
PropEnvCO2 = "ht:prop:env:co2"
PropEnvCpuload = "ht:prop:env:cpuload"
PropEnvDewpoint = "ht:prop:env:dewpoint"
PropEnvFuelFlowrate = "ht:prop:env:fuel:flowrate"
PropEnvFuelLevel = "ht:prop:env:fuel:level"
PropEnvHumidex = "ht:prop:env:humidex"
PropEnvHumidity = "ht:prop:env:humidity"
PropEnvLuminance = "ht:prop:env:luminance"
PropEnvPressure = "ht:prop:env:pressure"
PropEnvTemperature = "ht:prop:env:temperature"
PropEnvTimezone = "ht:prop:env:timezone"
PropEnvUV = "ht:prop:env:uv"
PropEnvVibration = "ht:prop:env:vibration"
PropEnvVolume = "ht:prop:env:volume"
PropEnvWaterFlowrate = "ht:prop:env:water:flowrate"
PropEnvWaterLevel = "ht:prop:env:water:level"
PropEnvWindHeading = "ht:prop:env:wind:heading"
PropEnvWindSpeed = "ht:prop:env:wind:speed"
PropLocation = "ht:prop:location"
PropLocationCity = "ht:prop:location:city"
PropLocationLatitude = "ht:prop:location:latitude"
PropLocationLongitude = "ht:prop:location:longitude"
PropLocationName = "ht:prop:location:name"
PropLocationStreet = "ht:prop:location:street"
PropLocationZipcode = "ht:prop:location:zipcode"
PropMedia = "ht:prop:media"
PropMediaMuted = "ht:prop:media:muted"
PropMediaPaused = "ht:prop:media:paused"
PropMediaPlaying = "ht:prop:media:playing"
PropMediaStation = "ht:prop:media:station"
PropMediaTrack = "ht:prop:media:track"
PropMediaVolume = "ht:prop:media:volume"
PropNet = "ht:prop:net"
PropNetAddress = "ht:prop:net:address"
PropNetConnection = "ht:prop:net:connection"
PropNetDomainname = "ht:prop:net:domainname"
PropNetGateway = "ht:prop:net:gateway"
PropNetHostname = "ht:prop:net:hostname"
PropNetIP4 = "ht:prop:net:ip4"
PropNetIP6 = "ht:prop:net:ip6"
PropNetLatency = "ht:prop:net:latency"
PropNetMAC = "ht:prop:net:mac"
PropNetMask = "ht:prop:net:mask"
PropNetPort = "ht:prop:net:port"
PropNetSignalstrength = "ht:prop:net:signalstrength"
PropNetSubnet = "ht:prop:net:subnet"
PropStatusOnOff = "ht:prop:status:onoff"
PropStatusOpenClosed = "ht:prop:status:openclosed"
PropStatusStartedStopped = "ht:prop:status:started-stopped"
PropStatusYesNo = "ht:prop:status:yes-no"
PropSwitch = "ht:prop:switch"
PropSwitchDimmer = "ht:prop:switch:dimmer"
PropSwitchLight = "ht:prop:switch:light"
PropSwitchLocked = "ht:prop:switch:locked"
PropSwitchOnOff = "ht:prop:switch:onoff"
# end of PropertyClasses

# PropertyClassesMap maps @type to symbol, title and description
PropertyClassesMap = {
  "ht:prop:electric:current": {"Symbol": "", "Title": "Current", "Description": "Electrical current"},
  "ht:prop:env": {"Symbol": "", "Title": "Environmental property", "Description": "Property of environmental sensor"},
  "ht:prop:env:airquality": {"Symbol": "", "Title": "Air quality", "Description": "Air quality level"},
  "ht:prop:location:longitude": {"Symbol": "", "Title": "Longitude", "Description": "Longitude geographic coordinate"},
  "ht:prop:device:description": {"Symbol": "", "Title": "Description", "Description": "Device product description"},
  "ht:prop:electric:poer": {"Symbol": "", "Title": "Power", "Description": "Electrical power being consumed"},
  "ht:prop:location:zipcode": {"Symbol": "", "Title": "Zip code", "Description": "Location ZIP code"},
  "ht:prop:net:gateway": {"Symbol": "", "Title": "Gateway", "Description": "Network gateway address"},
  "ht:prop:status:yes-no": {"Symbol": "", "Title": "Yes/No", "Description": "Status with yes or no value"},
  "ht:prop:device:firmwareversion": {"Symbol": "", "Title": "Firmware version", "Description": ""},
  "ht:prop:electric:overload": {"Symbol": "", "Title": "Overload protection", "Description": "Cut load on overload"},
  "ht:prop:env:co2": {"Symbol": "", "Title": "Carbon dioxide level", "Description": "Carbon dioxide level"},
  "ht:prop:env:fuel:flowrate": {"Symbol": "", "Title": "Fuel flow rate", "Description": ""},
  "ht:prop:media:muted": {"Symbol": "", "Title": "Muted", "Description": "Audio is muted"},
  "ht:prop:status:onoff": {"Symbol": "", "Title": "On/off status", "Description": ""},
  "ht:prop:switch:locked": {"Symbol": "", "Title": "Lock", "Description": "Electric lock status"},
  "ht:prop:device:battery": {"Symbol": "", "Title": "Battery level", "Description": "Device battery level"},
  "ht:prop:device:make": {"Symbol": "", "Title": "Make", "Description": "Device manufacturer"},
  "ht:prop:env:acceleration": {"Symbol": "", "Title": "Acceleration", "Description": ""},
  "ht:prop:env:humidity": {"Symbol": "", "Title": "Humidity", "Description": ""},
  "ht:prop:media:paused": {"Symbol": "", "Title": "Paused", "Description": "Media is paused"},
  "ht:prop:net:address": {"Symbol": "", "Title": "Address", "Description": "Network address"},
  "ht:prop:media:track": {"Symbol": "", "Title": "Track", "Description": "Selected A/V track"},
  "ht:prop:net:domainname": {"Symbol": "", "Title": "Domain name", "Description": "Domainname of the client"},
  "ht:prop:net:mac": {"Symbol": "", "Title": "MAC", "Description": "Hardware MAC address"},
  "ht:prop:electric:energy": {"Symbol": "", "Title": "Energy", "Description": "Electrical energy consumed"},
  "ht:prop:env:humidex": {"Symbol": "", "Title": "Humidex", "Description": ""},
  "ht:prop:env:uv": {"Symbol": "", "Title": "UV", "Description": ""},
  "ht:prop:env:volume": {"Symbol": "", "Title": "Volume", "Description": ""},
  "ht:prop:location:street": {"Symbol": "", "Title": "Street", "Description": "Street address"},
  "ht:prop:media:station": {"Symbol": "", "Title": "Station", "Description": "Selected radio station"},
  "ht:prop:device:model": {"Symbol": "", "Title": "Model", "Description": "Device model"},
  "ht:prop:device:pollinterval": {"Symbol": "", "Title": "Polling interval", "Description": "Interval to poll for updates"},
  "ht:prop:device:softwareversion": {"Symbol": "", "Title": "Software version", "Description": ""},
  "ht:prop:env:co": {"Symbol": "", "Title": "Carbon monoxide level", "Description": "Carbon monoxide level"},
  "ht:prop:env:dewpoint": {"Symbol": "", "Title": "Dew point", "Description": "Dew point temperature"},
  "ht:prop:location": {"Symbol": "", "Title": "Location", "Description": "General location information"},
  "ht:prop:location:city": {"Symbol": "", "Title": "City", "Description": "City name"},
  "ht:prop:net:ip6": {"Symbol": "", "Title": "IP6 address", "Description": "Device IP6 address"},
  "ht:prop:switch": {"Symbol": "", "Title": "Switch status", "Description": ""},
  "ht:prop:switch:light": {"Symbol": "", "Title": "Light switch", "Description": ""},
  "ht:prop:device": {"Symbol": "", "Title": "Device attributes", "Description": "Attributes describing a device"},
  "ht:prop:device:status": {"Symbol": "", "Title": "Status", "Description": "Device status; alive, awake, dead, sleeping"},
  "ht:prop:env:fuel:level": {"Symbol": "", "Title": "Fuel level", "Description": ""},
  "ht:prop:media:playing": {"Symbol": "", "Title": "Playing", "Description": "Media is playing"},
  "ht:prop:net:connection": {"Symbol": "", "Title": "Connection", "Description": "Connection status, connected, connecting, retrying, disconnected,..."},
  "ht:prop:switch:onoff": {"Symbol": "", "Title": "On/Off switch", "Description": ""},
  "ht:prop:env:pressure": {"Symbol": "", "Title": "Pressure", "Description": ""},
  "ht:prop:env:water:flowrate": {"Symbol": "", "Title": "Water flow rate", "Description": ""},
  "ht:prop:media": {"Symbol": "", "Title": "Media commands", "Description": "Control of media equipment"},
  "ht:prop:net:port": {"Symbol": "", "Title": "Port", "Description": "Network port"},
  "ht:prop:alarm:motion": {"Symbol": "", "Title": "Motion", "Description": "Motion detected"},
  "ht:prop:env:cpuload": {"Symbol": "", "Title": "CPU load level", "Description": "Device CPU load level"},
  "ht:prop:env:wind:heading": {"Symbol": "", "Title": "Wind heading", "Description": ""},
  "ht:prop:media:volume": {"Symbol": "", "Title": "Volume", "Description": "Media volume setting"},
  "ht:prop:env:temperature": {"Symbol": "", "Title": "Temperature", "Description": ""},
  "ht:prop:location:name": {"Symbol": "", "Title": "Location name", "Description": "Name of the location"},
  "ht:prop:device:hardwareversion": {"Symbol": "", "Title": "Hardware version", "Description": ""},
  "ht:prop:location:latitude": {"Symbol": "", "Title": "Latitude", "Description": "Latitude geographic coordinate"},
  "ht:prop:net:hostname": {"Symbol": "", "Title": "Hostname", "Description": "Hostname of the client"},
  "ht:prop:net:ip4": {"Symbol": "", "Title": "IP4 address", "Description": "Device IP4 address"},
  "ht:prop:net:latency": {"Symbol": "", "Title": "Network latency", "Description": "Delay between hub and client"},
  "ht:prop:status:started-stopped": {"Symbol": "", "Title": "Started/Stopped", "Description": "Started or stopped status"},
  "ht:prop:device:title": {"Symbol": "", "Title": "Title", "Description": "Device friendly title"},
  "ht:prop:env:vibration": {"Symbol": "", "Title": "Vibration", "Description": ""},
  "ht:prop:env:water:level": {"Symbol": "", "Title": "Water level", "Description": ""},
  "ht:prop:electric": {"Symbol": "", "Title": "Electrical properties", "Description": "General group of electrical properties"},
  "ht:prop:env:barometer": {"Symbol": "", "Title": "Atmospheric pressure", "Description": "Barometric pressure of the atmosphere"},
  "ht:prop:device:enabled-disabled": {"Symbol": "", "Title": "Enabled/Disabled", "Description": "Enabled or disabled state"},
  "ht:prop:electric:voltage": {"Symbol": "", "Title": "Voltage", "Description": "Electrical voltage potential"},
  "ht:prop:env:luminance": {"Symbol": "", "Title": "Luminance", "Description": ""},
  "ht:prop:env:wind:speed": {"Symbol": "", "Title": "Wind speed", "Description": ""},
  "ht:prop:net:subnet": {"Symbol": "", "Title": "Subnet", "Description": "Network subnet address. Example: 192.168.0.0"},
  "ht:prop:alarm:status": {"Symbol": "", "Title": "Alarm state", "Description": "Current alarm status"},
  "ht:prop:env:timezone": {"Symbol": "", "Title": "Timezone", "Description": ""},
  "ht:prop:net": {"Symbol": "", "Title": "Network properties", "Description": "General network properties"},
  "ht:prop:net:mask": {"Symbol": "", "Title": "Netmask", "Description": "Network mask. Example: 255.255.255.0 or 24/8"},
  "ht:prop:net:signalstrength": {"Symbol": "", "Title": "Signal strength", "Description": "Wireless signal strength"},
  "ht:prop:status:openclosed": {"Symbol": "", "Title": "Open/Closed status", "Description": ""},
  "ht:prop:switch:dimmer": {"Symbol": "", "Title": "Dimmer value", "Description": ""},
}


# type: ThingClasses
# version: 0.1
# generated: 19 Sep 24 19:02 PDT
# source: github.com/hiveot/hub/api/vocab/ht-thing-classes.yaml
# namespace: ht
ThingActuator = "ht:thing:actuator"
ThingActuatorAlarm = "ht:thing:actuator:alarm"
ThingActuatorBeacon = "ht:thing:actuator:beacon"
ThingActuatorDimmer = "ht:thing:actuator:dimmer"
ThingActuatorLight = "ht:thing:actuator:light"
ThingActuatorLock = "ht:thing:actuator:lock"
ThingActuatorMotor = "ht:thing:actuator:motor"
ThingActuatorOutput = "ht:thing:actuator:output"
ThingActuatorRanged = "ht:thing:actuator:ranged"
ThingActuatorRelay = "ht:thing:actuator:relay"
ThingActuatorSwitch = "ht:thing:actuator:switch"
ThingActuatorValve = "ht:thing:actuator:valve"
ThingActuatorValveFuel = "ht:thing:actuator:valve:fuel"
ThingActuatorValveWater = "ht:thing:actuator:valve:water"
ThingAppliance = "ht:thing:appliance"
ThingApplianceDishwasher = "ht:thing:appliance:dishwasher"
ThingApplianceDryer = "ht:thing:appliance:dryer"
ThingApplianceFreezer = "ht:thing:appliance:freezer"
ThingApplianceFridge = "ht:thing:appliance:fridge"
ThingApplianceWasher = "ht:thing:appliance:washer"
ThingComputer = "ht:thing:computer"
ThingComputerCellphone = "ht:thing:computer:cellphone"
ThingComputerEmbedded = "ht:thing:computer:embedded"
ThingComputerMemory = "ht:thing:computer:memory"
ThingComputerPC = "ht:thing:computer:pc"
ThingComputerPotsPhone = "ht:thing:computer:potsphone"
ThingComputerSatPhone = "ht:thing:computer:satphone"
ThingComputerTablet = "ht:thing:computer:tablet"
ThingComputerVoipPhone = "ht:thing:computer:voipphone"
ThingControl = "ht:thing:control"
ThingControlClimate = "ht:thing:control:climate"
ThingControlDimmer = "ht:thing:control:dimmer"
ThingControlIrrigation = "ht:thing:control:irrigation"
ThingControlJoystick = "ht:thing:control:joystick"
ThingControlKeypad = "ht:thing:control:keypad"
ThingControlPool = "ht:thing:control:pool"
ThingControlPushbutton = "ht:thing:control:pushbutton"
ThingControlSwitch = "ht:thing:control:switch"
ThingControlThermostat = "ht:thing:control:thermostat"
ThingControlToggle = "ht:thing:control:toggle"
ThingDevice = "ht:thing:device"
ThingDeviceBatteryMonitor = "ht:thing:device:battery:monitor"
ThingDeviceIndicator = "ht:thing:device:indicator"
ThingDeviceTime = "ht:thing:device:time"
ThingMedia = "ht:thing:media"
ThingMediaAmplifier = "ht:thing:media:amplifier"
ThingMediaCamera = "ht:thing:media:camera"
ThingMediaMicrophone = "ht:thing:media:microphone"
ThingMediaPlayer = "ht:thing:media:player"
ThingMediaRadio = "ht:thing:media:radio"
ThingMediaReceiver = "ht:thing:media:receiver"
ThingMediaSpeaker = "ht:thing:media:speaker"
ThingMediaTV = "ht:thing:media:tv"
ThingMeter = "ht:thing:meter"
ThingMeterElectric = "ht:thing:meter:electric"
ThingMeterElectricCurrent = "ht:thing:meter:electric:current"
ThingMeterElectricEnergy = "ht:thing:meter:electric:energy"
ThingMeterElectricPower = "ht:thing:meter:electric:power"
ThingMeterElectricVoltage = "ht:thing:meter:electric:voltage"
ThingMeterFuel = "ht:thing:meter:fuel"
ThingMeterFuelFlow = "ht:thing:meter:fuel:flow"
ThingMeterFuelLevel = "ht:thing:meter:fuel:level"
ThingMeterWater = "ht:thing:meter:water"
ThingMeterWaterConsumption = "ht:thing:meter:water:consumption"
ThingMeterWaterFlow = "ht:thing:meter:water:flow"
ThingMeterWaterLevel = "ht:thing:meter:water:level"
ThingMeterWind = "ht:thing:meter:wind"
ThingNet = "ht:thing:net"
ThingNetBluetooth = "ht:thing:net:bluetooth"
ThingNetGateway = "ht:thing:net:gateway"
ThingNetGatewayCoap = "ht:thing:net:gateway:coap"
ThingNetGatewayInsteon = "ht:thing:net:gateway:insteon"
ThingNetGatewayOnewire = "ht:thing:net:gateway:onewire"
ThingNetGatewayZigbee = "ht:thing:net:gateway:zigbee"
ThingNetGatewayZwave = "ht:thing:net:gateway:zwave"
ThingNetLora = "ht:thing:net:lora"
ThingNetLoraGateway = "ht:thing:net:lora:gw"
ThingNetLoraP2P = "ht:thing:net:lora:p2p"
ThingNetRouter = "ht:thing:net:router"
ThingNetSwitch = "ht:thing:net:switch"
ThingNetWifi = "ht:thing:net:wifi"
ThingNetWifiAp = "ht:thing:net:wifi:ap"
ThingSensor = "ht:thing:sensor"
ThingSensorEnvironment = "ht:thing:sensor:environment"
ThingSensorInput = "ht:thing:sensor:input"
ThingSensorMulti = "ht:thing:sensor:multi"
ThingSensorScale = "ht:thing:sensor:scale"
ThingSensorSecurity = "ht:thing:sensor:security"
ThingSensorSecurityDoorWindow = "ht:thing:sensor:security:doorwindow"
ThingSensorSecurityGlass = "ht:thing:sensor:security:glass"
ThingSensorSecurityMotion = "ht:thing:sensor:security:motion"
ThingSensorSmoke = "ht:thing:sensor:smoke"
ThingSensorSound = "ht:thing:sensor:sound"
ThingSensorThermometer = "ht:thing:sensor:thermometer"
ThingSensorWaterLeak = "ht:thing:sensor:water:leak"
ThingService = "ht:thing:service"
ThingServiceAdapter = "ht:thing:service:adapter"
ThingServiceAuth = "ht:thing:service:auth"
ThingServiceAutomation = "ht:thing:service:automation"
ThingServiceDirectory = "ht:thing:service:directory"
ThingServiceHistory = "ht:thing:service:history"
ThingServiceImage = "ht:thing:service:image"
ThingServiceSTT = "ht:thing:service:stt"
ThingServiceStore = "ht:thing:service:store"
ThingServiceTTS = "ht:thing:service:tts"
ThingServiceTranslation = "ht:thing:service:translation"
ThingServiceWeather = "ht:thing:service:weather"
ThingServiceWeatherCurrent = "ht:thing:service:weather:current"
ThingServiceWeatherForecast = "ht:thing:service:weather:forecast"
# end of ThingClasses

# ThingClassesMap maps @type to symbol, title and description
ThingClassesMap = {
  "ht:thing:actuator": {"Symbol": "", "Title": "Actuator", "Description": "Generic actuator"},
  "ht:thing:net:gateway:onewire": {"Symbol": "", "Title": "1-Wire gateway", "Description": "Gateway providing access to 1-wire devices"},
  "ht:thing:sensor:water:leak": {"Symbol": "", "Title": "Water leak detector", "Description": "Dedicated water leak detector"},
  "ht:thing:control:thermostat": {"Symbol": "", "Title": "Thermostat", "Description": "Thermostat HVAC control"},
  "ht:thing:meter:fuel:flow": {"Symbol": "", "Title": "Fuel flow rate", "Description": "Dedicated fuel flow rate metering device"},
  "ht:thing:meter:fuel:level": {"Symbol": "", "Title": "Fuel level", "Description": "Dedicated fuel level metering device"},
  "ht:thing:device": {"Symbol": "", "Title": "Device", "Description": "Device of unknown purpose"},
  "ht:thing:net:switch": {"Symbol": "", "Title": "Network switch", "Description": "Network switch to connect computer devices to the network"},
  "ht:thing:actuator:valve:water": {"Symbol": "", "Title": "Water valve", "Description": "Electric powered water valve"},
  "ht:thing:appliance:dryer": {"Symbol": "", "Title": "Dryer", "Description": "Clothing dryer"},
  "ht:thing:computer:memory": {"Symbol": "", "Title": "Memory", "Description": "Stand-alone memory device such as eeprom or iButtons"},
  "ht:thing:meter:electric:voltage": {"Symbol": "", "Title": "Voltage", "Description": "Electrical voltage meter"},
  "ht:thing:sensor:thermometer": {"Symbol": "", "Title": "Thermometer", "Description": "Environmental thermometer"},
  "ht:thing:control:keypad": {"Symbol": "", "Title": "Keypad", "Description": "Multi-key pad for command input"},
  "ht:thing:control:pool": {"Symbol": "", "Title": "Pool control", "Description": "Device for controlling pool settings"},
  "ht:thing:meter": {"Symbol": "", "Title": "Meter", "Description": "General metering device"},
  "ht:thing:meter:wind": {"Symbol": "", "Title": "Wind", "Description": "Dedicated wind meter"},
  "ht:thing:control:climate": {"Symbol": "", "Title": "Climate control", "Description": "Device for controlling climate of a space"},
  "ht:thing:meter:electric:energy": {"Symbol": "", "Title": "Electric energy", "Description": "Electrical energy meter"},
  "ht:thing:net:wifi:ap": {"Symbol": "", "Title": "Wifi access point", "Description": "Wireless access point for IP networks"},
  "ht:thing:sensor:security": {"Symbol": "", "Title": "Security", "Description": "Generic security sensor"},
  "ht:thing:media:tv": {"Symbol": "", "Title": "TV", "Description": "Network connected television"},
  "ht:thing:net:router": {"Symbol": "", "Title": "Network router", "Description": "IP ThingNetwork router providing access to other IP networks"},
  "ht:thing:appliance": {"Symbol": "", "Title": "Appliance", "Description": "Appliance to accomplish a particular task for occupant use"},
  "ht:thing:meter:electric": {"Symbol": "", "Title": "", "Description": ""},
  "ht:thing:meter:water:level": {"Symbol": "", "Title": "Water level", "Description": "Dedicated water level meter"},
  "ht:thing:net:lora": {"Symbol": "", "Title": "LoRa network device", "Description": "Generic Long Range network protocol device"},
  "ht:thing:sensor:smoke": {"Symbol": "", "Title": "Smoke detector", "Description": ""},
  "ht:thing:service:adapter": {"Symbol": "", "Title": "Protocol adapter", "Description": "Protocol adapter/binding for integration with another protocol"},
  "ht:thing:actuator:valve": {"Symbol": "", "Title": "Valve", "Description": "Electric powered valve for fluids or gas"},
  "ht:thing:media:microphone": {"Symbol": "", "Title": "Microphone", "Description": "Microphone for capturing audio"},
  "ht:thing:media:receiver": {"Symbol": "", "Title": "Receiver", "Description": "Audio/video receiver and player"},
  "ht:thing:net:gateway:zigbee": {"Symbol": "", "Title": "Zigbee gateway", "Description": "Gateway providing access to Zigbee devices"},
  "ht:thing:sensor:input": {"Symbol": "", "Title": "Input sensor", "Description": "General purpose electrical input sensor"},
  "ht:thing:service:directory": {"Symbol": "", "Title": "Directory service", "Description": ""},
  "ht:thing:service:translation": {"Symbol": "", "Title": "Language translation service", "Description": ""},
  "ht:thing:appliance:dishwasher": {"Symbol": "", "Title": "Dishwasher", "Description": "Dishwasher"},
  "ht:thing:device:time": {"Symbol": "", "Title": "Clock", "Description": "Time tracking device such as clocks and time chips"},
  "ht:thing:service:auth": {"Symbol": "", "Title": "Authentication service", "Description": ""},
  "ht:thing:service:automation": {"Symbol": "", "Title": "Automation service", "Description": ""},
  "ht:thing:service:image": {"Symbol": "", "Title": "Image classification", "Description": ""},
  "ht:thing:actuator:light": {"Symbol": "", "Title": "Light", "Description": "Smart LED or other light"},
  "ht:thing:actuator:relay": {"Symbol": "", "Title": "Relay", "Description": "Generic relay electrical switch"},
  "ht:thing:control:joystick": {"Symbol": "", "Title": "Joystick", "Description": "Flight control stick"},
  "ht:thing:net:lora:gw": {"Symbol": "", "Title": "LoRaWAN gateway", "Description": "Gateway providing access to LoRa devices"},
  "ht:thing:actuator:valve:fuel": {"Symbol": "", "Title": "Fuel valve", "Description": "Electric powered fuel valve"},
  "ht:thing:control:dimmer": {"Symbol": "", "Title": "Dimmer", "Description": "Light dimmer input device"},
  "ht:thing:sensor:scale": {"Symbol": "", "Title": "Scale", "Description": "Electronic weigh scale"},
  "ht:thing:service:tts": {"Symbol": "", "Title": "Text to speech", "Description": ""},
  "ht:thing:control": {"Symbol": "", "Title": "Input controller", "Description": "Generic input controller"},
  "ht:thing:control:switch": {"Symbol": "", "Title": "Input switch", "Description": "On or off switch input control"},
  "ht:thing:net:gateway:coap": {"Symbol": "", "Title": "CoAP gateway", "Description": "Gateway providing access to CoAP devices"},
  "ht:thing:sensor:environment": {"Symbol": "", "Title": "Environmental sensor", "Description": "Environmental sensor with one or more features such as temperature, humidity, etc"},
  "ht:thing:sensor:security:motion": {"Symbol": "", "Title": "Motion sensor", "Description": "Dedicated security sensor detecting motion"},
  "ht:thing:service:weather": {"Symbol": "", "Title": "Weather service", "Description": "General weather service"},
  "ht:thing:actuator:dimmer": {"Symbol": "", "Title": "Dimmer", "Description": "Light dimmer"},
  "ht:thing:control:pushbutton": {"Symbol": "", "Title": "Momentary switch", "Description": "Momentary push button control input"},
  "ht:thing:meter:fuel": {"Symbol": "", "Title": "Fuel metering device", "Description": "General fuel metering device"},
  "ht:thing:actuator:motor": {"Symbol": "", "Title": "Motor", "Description": "Motor driven actuator, such as garage door, blinds, tv lifts"},
  "ht:thing:sensor": {"Symbol": "", "Title": "Sensor", "Description": "Generic sensor device"},
  "ht:thing:computer:voipphone": {"Symbol": "", "Title": "VoIP Phone", "Description": "Voice over IP phone"},
  "ht:thing:media:speaker": {"Symbol": "", "Title": "Connected speakers", "Description": "Network connected speakers"},
  "ht:thing:meter:electric:current": {"Symbol": "", "Title": "Electric current", "Description": "Electrical current meter"},
  "ht:thing:net:gateway:insteon": {"Symbol": "", "Title": "Insteon gateway", "Description": "Gateway providing access to Insteon devices"},
  "ht:thing:sensor:security:glass": {"Symbol": "", "Title": "Glass sensor", "Description": "Dedicated sensor for detecting breaking of glass"},
  "ht:thing:computer": {"Symbol": "", "Title": "Computing Device", "Description": "General purpose computing device"},
  "ht:thing:computer:satphone": {"Symbol": "", "Title": "Satellite phone", "Description": ""},
  "ht:thing:service:weather:current": {"Symbol": "", "Title": "Current weather", "Description": ""},
  "ht:thing:appliance:freezer": {"Symbol": "", "Title": "Freezer", "Description": "Refrigerator freezer"},
  "ht:thing:device:battery:monitor": {"Symbol": "", "Title": "Battery Monitor", "Description": "Battery monitor and charge controller"},
  "ht:thing:service:stt": {"Symbol": "", "Title": "Speech to text", "Description": ""},
  "ht:thing:service:weather:forecast": {"Symbol": "", "Title": "Weather forecast", "Description": ""},
  "ht:thing:computer:cellphone": {"Symbol": "", "Title": "Cell Phone", "Description": "Cellular phone"},
  "ht:thing:sensor:sound": {"Symbol": "", "Title": "Sound detector", "Description": ""},
  "ht:thing:computer:embedded": {"Symbol": "", "Title": "Embedded System", "Description": "Embedded computing device"},
  "ht:thing:net": {"Symbol": "", "Title": "Network device", "Description": "Generic network device"},
  "ht:thing:net:wifi": {"Symbol": "", "Title": "Wifi device", "Description": "Generic wifi device"},
  "ht:thing:meter:water": {"Symbol": "", "Title": "Water metering device", "Description": "General water metering device"},
  "ht:thing:sensor:multi": {"Symbol": "", "Title": "Multi sensor", "Description": "Sense multiple inputs"},
  "ht:thing:service:history": {"Symbol": "", "Title": "History service", "Description": ""},
  "ht:thing:net:gateway": {"Symbol": "", "Title": "Gateway", "Description": "Generic gateway device providing access to other devices"},
  "ht:thing:actuator:switch": {"Symbol": "", "Title": "Switch", "Description": "An electric powered on/off switch for powering circuits"},
  "ht:thing:media": {"Symbol": "", "Title": "A/V media", "Description": "Generic device for audio/video media record or playback"},
  "ht:thing:media:camera": {"Symbol": "", "Title": "Camera", "Description": "Video camera"},
  "ht:thing:media:player": {"Symbol": "", "Title": "Media player", "Description": "CD/DVD/Blueray/USB player of recorded media"},
  "ht:thing:service": {"Symbol": "", "Title": "Service", "Description": "General service for processing data and offering features of interest"},
  "ht:thing:control:toggle": {"Symbol": "", "Title": "Toggle switch", "Description": "Toggle switch input control"},
  "ht:thing:actuator:lock": {"Symbol": "", "Title": "Lock", "Description": "Electronic door lock"},
  "ht:thing:service:store": {"Symbol": "", "Title": "Data storage", "Description": ""},
  "ht:thing:actuator:ranged": {"Symbol": "", "Title": "Ranged actuator", "Description": "Generic ranged actuator with a set point"},
  "ht:thing:appliance:washer": {"Symbol": "", "Title": "Washer", "Description": "Clothing washer"},
  "ht:thing:computer:pc": {"Symbol": "", "Title": "PC/Laptop", "Description": "Personal computer/laptop"},
  "ht:thing:meter:water:flow": {"Symbol": "", "Title": "Water flow", "Description": "Dedicated water flow-rate meter"},
  "ht:thing:meter:water:consumption": {"Symbol": "", "Title": "Water consumption meter", "Description": "Water consumption meter"},
  "ht:thing:actuator:alarm": {"Symbol": "", "Title": "Alarm", "Description": "Siren or light alarm"},
  "ht:thing:actuator:beacon": {"Symbol": "", "Title": "Beacon", "Description": "Location beacon"},
  "ht:thing:actuator:output": {"Symbol": "", "Title": "Output", "Description": "General purpose electrical output signal"},
  "ht:thing:appliance:fridge": {"Symbol": "", "Title": "Fridge", "Description": "Refrigerator appliance"},
  "ht:thing:media:amplifier": {"Symbol": "", "Title": "Audio amplifier", "Description": "Audio amplifier with volume controls"},
  "ht:thing:media:radio": {"Symbol": "", "Title": "Radio", "Description": "AM or FM radio receiver"},
  "ht:thing:meter:electric:power": {"Symbol": "", "Title": "Electrical Power", "Description": "Electrical power meter"},
  "ht:thing:net:bluetooth": {"Symbol": "", "Title": "Bluetooth", "Description": "Bluetooth radio"},
  "ht:thing:control:irrigation": {"Symbol": "", "Title": "Irrigation control", "Description": "Device for control of an irrigation system"},
  "ht:thing:net:gateway:zwave": {"Symbol": "", "Title": "ZWave gateway", "Description": "Gateway providing access to ZWave devices"},
  "ht:thing:sensor:security:doorwindow": {"Symbol": "", "Title": "Door/Window sensor", "Description": "Dedicated door/window opening security sensor"},
  "ht:thing:computer:potsphone": {"Symbol": "", "Title": "Land Line", "Description": "Plain Old Telephone System, aka landline"},
  "ht:thing:computer:tablet": {"Symbol": "", "Title": "Tablet", "Description": "Tablet computer"},
  "ht:thing:device:indicator": {"Symbol": "", "Title": "Indicator", "Description": "Visual or audio indicator device"},
  "ht:thing:net:lora:p2p": {"Symbol": "", "Title": "LoRa P2P", "Description": "LoRa Peer-to-peer network device"},
}


# type: UnitClasses
# version: 0.1
# generated: 19 Sep 24 19:02 PDT
# source: github.com/hiveot/hub/api/vocab/ht-unit-classes.yaml
# namespace: ht
UnitAmpere = "ht:unit:ampere"
UnitCandela = "ht:unit:candela"
UnitCelcius = "ht:unit:celcius"
UnitCount = "ht:unit:count"
UnitDegree = "ht:unit:degree"
UnitFahrenheit = "ht:unit:fahrenheit"
UnitFoot = "ht:unit:foot"
UnitGallon = "ht:unit:gallon"
UnitKelvin = "ht:unit:kelvin"
UnitKilogram = "ht:unit:kilogram"
UnitKilometerPerHour = "ht:unit:kph"
UnitKilowattHour = "ht:unit:kilowatthour"
UnitLiter = "ht:unit:liter"
UnitLumen = "ht:unit:lumen"
UnitLux = "ht:unit:lux"
UnitMercury = "ht:unit:mercury"
UnitMeter = "ht:unit:meter"
UnitMeterPerSecond = "ht:unit:meterspersecond"
UnitMilesPerHour = "ht:unit:milesperhour"
UnitMilliSecond = "ht:unit:millisecond"
UnitMillibar = "ht:unit:millibar"
UnitMole = "ht:unit:mole"
UnitPSI = "ht:unit:psi"
UnitPascal = "ht:unit:pascal"
UnitPercent = "ht:unit:percent"
UnitPound = "ht:unit:pound"
UnitPpm = "ht:unit:ppm"
UnitRadian = "ht:unit:radian"
UnitSecond = "ht:unit:second"
UnitVolt = "ht:unit:volt"
UnitWatt = "ht:unit:watt"
# end of UnitClasses

# UnitClassesMap maps @type to symbol, title and description
UnitClassesMap = {
  "ht:unit:millibar": {"Symbol": "mbar", "Title": "millibar", "Description": "Metric unit of pressure. 1/1000th of a bar. Equal to 100 pascals. Amount of force it takes to move an object weighing a gram, one centimeter in one second."},
  "ht:unit:ppm": {"Symbol": "ppm", "Title": "PPM", "Description": "Parts per million"},
  "ht:unit:second": {"Symbol": "s", "Title": "Second", "Description": "SI unit of time based on caesium frequency"},
  "ht:unit:fahrenheit": {"Symbol": "F", "Title": "Fahrenheit", "Description": "Temperature in Fahrenheit"},
  "ht:unit:meterspersecond": {"Symbol": "m/s", "Title": "Meters per second", "Description": "SI unit of speed in meters per second"},
  "ht:unit:mole": {"Symbol": "mol", "Title": "Mole", "Description": "SI unit of measurement for amount of substance. Eg, molecules."},
  "ht:unit:pound": {"Symbol": "lbs", "Title": "Pound", "Description": "Imperial unit of weight. Equivalent to 0.453592 Kg. 1 Kg is 2.205 lbs"},
  "ht:unit:milesperhour": {"Symbol": "mph", "Title": "Miles per hour", "Description": "Speed in miles per hour"},
  "ht:unit:psi": {"Symbol": "PSI", "Title": "PSI", "Description": "Unit of pressure. Pounds of force per square inch. 1PSI equals 6984 Pascals."},
  "ht:unit:volt": {"Symbol": "V", "Title": "Volt", "Description": "SI unit of electric potential; Energy consumption of 1 joule per electric charge of one coulomb"},
  "ht:unit:ampere": {"Symbol": "A", "Title": "Ampere", "Description": "Electrical current in Amperes based on the elementary charge flow per second"},
  "ht:unit:count": {"Symbol": "(N)", "Title": "Count", "Description": ""},
  "ht:unit:degree": {"Symbol": "degree", "Title": "Degree", "Description": "Angle in 0-360 degrees"},
  "ht:unit:kelvin": {"Symbol": "K", "Title": "Kelvin", "Description": "SI unit of thermodynamic temperature. 0 K represents absolute zero, the absence of all heat. 0 C equals +273.15K"},
  "ht:unit:meter": {"Symbol": "m", "Title": "Meter", "Description": "Distance in meters. 1m=c/299792458"},
  "ht:unit:millisecond": {"Symbol": "ms", "Title": "millisecond", "Description": "Unit of time in milli-seconds. Equal to 1/1000 of a second."},
  "ht:unit:watt": {"Symbol": "W", "Title": "Watt", "Description": "SI unit of power. Equal to 1 joule per second; or work performed when a current of 1 ampere flows across an electric potential of one volt."},
  "ht:unit:celcius": {"Symbol": "C", "Title": "Celcius", "Description": "Temperature in Celcius"},
  "ht:unit:foot": {"Symbol": "ft", "Title": "Foot", "Description": "Imperial unit of distance. 1 foot equals 0.3048 meters"},
  "ht:unit:lumen": {"Symbol": "lm", "Title": "Lumen", "Description": "SI unit luminous flux. Measure of perceived power of visible light. 1lm = 1 cd steradian"},
  "ht:unit:percent": {"Symbol": "%", "Title": "Percent", "Description": "Fractions of 100"},
  "ht:unit:gallon": {"Symbol": "gl", "Title": "Gallon", "Description": "Unit of volume. 1 Imperial gallon is 4.54609 liters. 1 US liquid gallon is 3.78541 liters. 1 US dry gallon is 4.405 liters. "},
  "ht:unit:liter": {"Symbol": "l", "Title": "Liter", "Description": "SI unit of volume equivalent to 1 cubic decimeter."},
  "ht:unit:lux": {"Symbol": "lx", "Title": "Lux", "Description": "SI unit illuminance. Equal to 1 lumen per square meter."},
  "ht:unit:mercury": {"Symbol": "Hg", "Title": "Mercury", "Description": "Unit of atmospheric pressure in the United States. 1 Hg equals 33.8639 mbar."},
  "ht:unit:kilogram": {"Symbol": "kg", "Title": "Kilogram", "Description": ""},
  "ht:unit:kilowatthour": {"Symbol": "kWh", "Title": "Kilowatt-hour", "Description": "non-SI unit of energy equivalent to 3.6 megajoules."},
  "ht:unit:pascal": {"Symbol": "Pa", "Title": "Pascal", "Description": "SI unit of pressure. Equal to 1 newton of force applied over 1 square meter."},
  "ht:unit:radian": {"Symbol": "", "Title": "Radian", "Description": "Angle in 0-2pi"},
  "ht:unit:candela": {"Symbol": "cd", "Title": "Candela", "Description": "SI unit of luminous intensity in a given direction. Roughly the same brightness as the common candle."},
  "ht:unit:kph": {"Symbol": "kph", "Title": "Km per hour", "Description": "Speed in kilometers per hour"},
}
