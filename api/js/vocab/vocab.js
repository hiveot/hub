// Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions
// DO NOT EDIT. This file is generated and changes will be overwritten

// type: WoTVocab
// version: 0.1
// generated: 21 Feb 25 21:45 PST
// source: github.com/hiveot/hub/api/vocab/wot-vocab.yaml
// description: WoT vocabulary definition. See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
export const OpCancelAction = "cancelaction"
export const OpInvokeAction = "invokeaction"
export const OpObserveAllProperties = "observeallproperties"
export const OpObserveProperty = "observeproperty"
export const OpQueryAction = "queryaction"
export const OpQueryAllActions = "queryallactions"
export const OpReadAllProperties = "readallproperties"
export const OpReadProperty = "readproperty"
export const OpSubscribeAllEvents = "subscribeallevents"
export const OpSubscribeEvent = "subscribeevent"
export const OpUnobserveAllProperties = "unobserveallproperties"
export const OpUnobserveProperty = "unobserveroperty"
export const OpUnsubscribeAllEvents = "unsubscribeallevents"
export const OpUnsubscribeEvent = "unsubscribeevent"
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

// type: ThingClasses
// version: 0.1
// generated: 21 Feb 25 21:45 PST
// source: github.com/hiveot/hub/api/vocab/ht-thing-classes.yaml
// namespace: hiveot
export const ThingActuator = "hiveot:thing:actuator";
export const ThingActuatorAlarm = "hiveot:thing:actuator:alarm";
export const ThingActuatorBeacon = "hiveot:thing:actuator:beacon";
export const ThingActuatorDimmer = "hiveot:thing:actuator:dimmer";
export const ThingActuatorLight = "hiveot:thing:actuator:light";
export const ThingActuatorLock = "hiveot:thing:actuator:lock";
export const ThingActuatorMotor = "hiveot:thing:actuator:motor";
export const ThingActuatorOutput = "hiveot:thing:actuator:output";
export const ThingActuatorRanged = "hiveot:thing:actuator:ranged";
export const ThingActuatorRelay = "hiveot:thing:actuator:relay";
export const ThingActuatorSwitch = "hiveot:thing:actuator:switch";
export const ThingActuatorValve = "hiveot:thing:actuator:valve";
export const ThingActuatorValveFuel = "hiveot:thing:actuator:valve:fuel";
export const ThingActuatorValveWater = "hiveot:thing:actuator:valve:water";
export const ThingAppliance = "hiveot:thing:appliance";
export const ThingApplianceDishwasher = "hiveot:thing:appliance:dishwasher";
export const ThingApplianceDryer = "hiveot:thing:appliance:dryer";
export const ThingApplianceFreezer = "hiveot:thing:appliance:freezer";
export const ThingApplianceFridge = "hiveot:thing:appliance:fridge";
export const ThingApplianceWasher = "hiveot:thing:appliance:washer";
export const ThingComputer = "hiveot:thing:computer";
export const ThingComputerCellphone = "hiveot:thing:computer:cellphone";
export const ThingComputerEmbedded = "hiveot:thing:computer:embedded";
export const ThingComputerMemory = "hiveot:thing:computer:memory";
export const ThingComputerPC = "hiveot:thing:computer:pc";
export const ThingComputerPotsPhone = "hiveot:thing:computer:potsphone";
export const ThingComputerSatPhone = "hiveot:thing:computer:satphone";
export const ThingComputerTablet = "hiveot:thing:computer:tablet";
export const ThingComputerVoipPhone = "hiveot:thing:computer:voipphone";
export const ThingControl = "hiveot:thing:control";
export const ThingControlClimate = "hiveot:thing:control:climate";
export const ThingControlDimmer = "hiveot:thing:control:dimmer";
export const ThingControlIrrigation = "hiveot:thing:control:irrigation";
export const ThingControlJoystick = "hiveot:thing:control:joystick";
export const ThingControlKeypad = "hiveot:thing:control:keypad";
export const ThingControlPool = "hiveot:thing:control:pool";
export const ThingControlPushbutton = "hiveot:thing:control:pushbutton";
export const ThingControlSwitch = "hiveot:thing:control:switch";
export const ThingControlThermostat = "hiveot:thing:control:thermostat";
export const ThingControlToggle = "hiveot:thing:control:toggle";
export const ThingDevice = "hiveot:thing:device";
export const ThingDeviceBatteryMonitor = "hiveot:thing:device:battery:monitor";
export const ThingDeviceIndicator = "hiveot:thing:device:indicator";
export const ThingDeviceTime = "hiveot:thing:device:time";
export const ThingMedia = "hiveot:thing:media";
export const ThingMediaAmplifier = "hiveot:thing:media:amplifier";
export const ThingMediaCamera = "hiveot:thing:media:camera";
export const ThingMediaMicrophone = "hiveot:thing:media:microphone";
export const ThingMediaPlayer = "hiveot:thing:media:player";
export const ThingMediaRadio = "hiveot:thing:media:radio";
export const ThingMediaReceiver = "hiveot:thing:media:receiver";
export const ThingMediaSpeaker = "hiveot:thing:media:speaker";
export const ThingMediaTV = "hiveot:thing:media:tv";
export const ThingMeter = "hiveot:thing:meter";
export const ThingMeterElectric = "hiveot:thing:meter:electric";
export const ThingMeterElectricCurrent = "hiveot:thing:meter:electric:current";
export const ThingMeterElectricEnergy = "hiveot:thing:meter:electric:energy";
export const ThingMeterElectricPower = "hiveot:thing:meter:electric:power";
export const ThingMeterElectricVoltage = "hiveot:thing:meter:electric:voltage";
export const ThingMeterFuel = "hiveot:thing:meter:fuel";
export const ThingMeterFuelFlow = "hiveot:thing:meter:fuel:flow";
export const ThingMeterFuelLevel = "hiveot:thing:meter:fuel:level";
export const ThingMeterWater = "hiveot:thing:meter:water";
export const ThingMeterWaterConsumption = "hiveot:thing:meter:water:consumption";
export const ThingMeterWaterFlow = "hiveot:thing:meter:water:flow";
export const ThingMeterWaterLevel = "hiveot:thing:meter:water:level";
export const ThingMeterWind = "hiveot:thing:meter:wind";
export const ThingNet = "hiveot:thing:net";
export const ThingNetBluetooth = "hiveot:thing:net:bluetooth";
export const ThingNetGateway = "hiveot:thing:net:gateway";
export const ThingNetGatewayCoap = "hiveot:thing:net:gateway:coap";
export const ThingNetGatewayInsteon = "hiveot:thing:net:gateway:insteon";
export const ThingNetGatewayOnewire = "hiveot:thing:net:gateway:onewire";
export const ThingNetGatewayZigbee = "hiveot:thing:net:gateway:zigbee";
export const ThingNetGatewayZwave = "hiveot:thing:net:gateway:zwave";
export const ThingNetLora = "hiveot:thing:net:lora";
export const ThingNetLoraGateway = "hiveot:thing:net:lora:gw";
export const ThingNetLoraP2P = "hiveot:thing:net:lora:p2p";
export const ThingNetRouter = "hiveot:thing:net:router";
export const ThingNetSwitch = "hiveot:thing:net:switch";
export const ThingNetWifi = "hiveot:thing:net:wifi";
export const ThingNetWifiAp = "hiveot:thing:net:wifi:ap";
export const ThingSensor = "hiveot:thing:sensor";
export const ThingSensorEnvironment = "hiveot:thing:sensor:environment";
export const ThingSensorInput = "hiveot:thing:sensor:input";
export const ThingSensorMulti = "hiveot:thing:sensor:multi";
export const ThingSensorScale = "hiveot:thing:sensor:scale";
export const ThingSensorSecurity = "hiveot:thing:sensor:security";
export const ThingSensorSecurityDoorWindow = "hiveot:thing:sensor:security:doorwindow";
export const ThingSensorSecurityGlass = "hiveot:thing:sensor:security:glass";
export const ThingSensorSecurityMotion = "hiveot:thing:sensor:security:motion";
export const ThingSensorSmoke = "hiveot:thing:sensor:smoke";
export const ThingSensorSound = "hiveot:thing:sensor:sound";
export const ThingSensorThermometer = "hiveot:thing:sensor:thermometer";
export const ThingSensorWaterLeak = "hiveot:thing:sensor:water:leak";
export const ThingService = "hiveot:thing:service";
// end of ThingClasses

// ThingClassesMap maps @type to symbol, title and description
export const ThingClassesMap = {
  "hiveot:thing:actuator:ranged": {Symbol: "", Title: "Ranged actuator", Description: "Generic ranged actuator with a set point"},
  "hiveot:thing:computer:satphone": {Symbol: "", Title: "Satellite phone", Description: ""},
  "hiveot:thing:net": {Symbol: "", Title: "Network device", Description: "Generic network device"},
  "hiveot:thing:computer:embedded": {Symbol: "", Title: "Embedded System", Description: "Embedded computing device"},
  "hiveot:thing:control:joystick": {Symbol: "", Title: "Joystick", Description: "Flight control stick"},
  "hiveot:thing:device:time": {Symbol: "", Title: "Clock", Description: "Time tracking device such as clocks and time chips"},
  "hiveot:thing:media:speaker": {Symbol: "", Title: "Connected speakers", Description: "Network connected speakers"},
  "hiveot:thing:meter:fuel:level": {Symbol: "", Title: "Fuel level", Description: "Dedicated fuel level metering device"},
  "hiveot:thing:sensor:input": {Symbol: "", Title: "Input sensor", Description: "General purpose electrical input sensor"},
  "hiveot:thing:meter:water:flow": {Symbol: "", Title: "Water flow", Description: "Dedicated water flow-rate meter"},
  "hiveot:thing:net:lora:p2p": {Symbol: "", Title: "LoRa P2P", Description: "LoRa Peer-to-peer network device"},
  "hiveot:thing:actuator:beacon": {Symbol: "", Title: "Beacon", Description: "Location beacon"},
  "hiveot:thing:actuator:lock": {Symbol: "", Title: "Lock", Description: "Electronic door lock"},
  "hiveot:thing:appliance:dryer": {Symbol: "", Title: "Dryer", Description: "Clothing dryer"},
  "hiveot:thing:appliance:washer": {Symbol: "", Title: "Washer", Description: "Clothing washer"},
  "hiveot:thing:computer:cellphone": {Symbol: "", Title: "Cell Phone", Description: "Cellular phone"},
  "hiveot:thing:meter:electric:power": {Symbol: "", Title: "Electrical Power", Description: "Electrical power meter"},
  "hiveot:thing:sensor:security:doorwindow": {Symbol: "", Title: "Door/Window sensor", Description: "Dedicated door/window opening security sensor"},
  "hiveot:thing:sensor:sound": {Symbol: "", Title: "Sound detector", Description: ""},
  "hiveot:thing:sensor:thermometer": {Symbol: "", Title: "Thermometer", Description: "Environmental thermometer"},
  "hiveot:thing:actuator:alarm": {Symbol: "", Title: "Alarm", Description: "Siren or light alarm"},
  "hiveot:thing:net:bluetooth": {Symbol: "", Title: "Bluetooth", Description: "Bluetooth radio"},
  "hiveot:thing:sensor:multi": {Symbol: "", Title: "Multi sensor", Description: "Sense multiple inputs"},
  "hiveot:thing:sensor:security:motion": {Symbol: "", Title: "Motion sensor", Description: "Dedicated security sensor detecting motion"},
  "hiveot:thing:actuator": {Symbol: "", Title: "Actuator", Description: "Generic actuator"},
  "hiveot:thing:media:microphone": {Symbol: "", Title: "Microphone", Description: "Microphone for capturing audio"},
  "hiveot:thing:meter:electric:current": {Symbol: "", Title: "Electric current", Description: "Electrical current meter"},
  "hiveot:thing:net:router": {Symbol: "", Title: "Network router", Description: "IP ThingNetwork router providing access to other IP networks"},
  "hiveot:thing:service": {Symbol: "", Title: "Service", Description: "General service for processing data and offering features of interest"},
  "hiveot:thing:sensor:water:leak": {Symbol: "", Title: "Water leak detector", Description: "Dedicated water leak detector"},
  "hiveot:thing:control": {Symbol: "", Title: "Input controller", Description: "Generic input controller"},
  "hiveot:thing:meter:electric:voltage": {Symbol: "", Title: "Voltage", Description: "Electrical voltage meter"},
  "hiveot:thing:meter:fuel:flow": {Symbol: "", Title: "Fuel flow rate", Description: "Dedicated fuel flow rate metering device"},
  "hiveot:thing:net:gateway:coap": {Symbol: "", Title: "CoAP gateway", Description: "Gateway providing access to CoAP devices"},
  "hiveot:thing:net:wifi": {Symbol: "", Title: "Wifi device", Description: "Generic wifi device"},
  "hiveot:thing:net:wifi:ap": {Symbol: "", Title: "Wifi access point", Description: "Wireless access point for IP networks"},
  "hiveot:thing:control:dimmer": {Symbol: "", Title: "Dimmer", Description: "Light dimmer input device"},
  "hiveot:thing:control:keypad": {Symbol: "", Title: "Keypad", Description: "Multi-key pad for command input"},
  "hiveot:thing:device": {Symbol: "", Title: "Device", Description: "Device of unknown purpose"},
  "hiveot:thing:media:receiver": {Symbol: "", Title: "Receiver", Description: "Audio/video receiver and player"},
  "hiveot:thing:sensor:environment": {Symbol: "", Title: "Environmental sensor", Description: "Environmental sensor with one or more features such as temperature, humidity, etc"},
  "hiveot:thing:meter:water": {Symbol: "", Title: "Water metering device", Description: "General water metering device"},
  "hiveot:thing:net:gateway:insteon": {Symbol: "", Title: "Insteon gateway", Description: "Gateway providing access to Insteon devices"},
  "hiveot:thing:net:lora": {Symbol: "", Title: "LoRa network device", Description: "Generic Long Range network protocol device"},
  "hiveot:thing:sensor:smoke": {Symbol: "", Title: "Smoke detector", Description: ""},
  "hiveot:thing:appliance:dishwasher": {Symbol: "", Title: "Dishwasher", Description: "Dishwasher"},
  "hiveot:thing:control:pushbutton": {Symbol: "", Title: "Momentary switch", Description: "Momentary push button control input"},
  "hiveot:thing:media:radio": {Symbol: "", Title: "Radio", Description: "AM or FM radio receiver"},
  "hiveot:thing:meter:fuel": {Symbol: "", Title: "Fuel metering device", Description: "General fuel metering device"},
  "hiveot:thing:net:gateway:zwave": {Symbol: "", Title: "ZWave gateway", Description: "Gateway providing access to ZWave devices"},
  "hiveot:thing:actuator:dimmer": {Symbol: "", Title: "Dimmer", Description: "Light dimmer"},
  "hiveot:thing:computer:potsphone": {Symbol: "", Title: "Land Line", Description: "Plain Old Telephone System, aka landline"},
  "hiveot:thing:media:amplifier": {Symbol: "", Title: "Audio amplifier", Description: "Audio amplifier with volume controls"},
  "hiveot:thing:media:camera": {Symbol: "", Title: "Camera", Description: "Video camera"},
  "hiveot:thing:media:player": {Symbol: "", Title: "Media player", Description: "CD/DVD/Blueray/USB player of recorded media"},
  "hiveot:thing:actuator:valve:fuel": {Symbol: "", Title: "Fuel valve", Description: "Electric powered fuel valve"},
  "hiveot:thing:appliance:freezer": {Symbol: "", Title: "Freezer", Description: "Refrigerator freezer"},
  "hiveot:thing:computer": {Symbol: "", Title: "Computing Device", Description: "General purpose computing device"},
  "hiveot:thing:computer:tablet": {Symbol: "", Title: "Tablet", Description: "Tablet computer"},
  "hiveot:thing:control:irrigation": {Symbol: "", Title: "Irrigation control", Description: "Device for control of an irrigation system"},
  "hiveot:thing:net:lora:gw": {Symbol: "", Title: "LoRaWAN gateway", Description: "Gateway providing access to LoRa devices"},
  "hiveot:thing:actuator:valve:water": {Symbol: "", Title: "Water valve", Description: "Electric powered water valve"},
  "hiveot:thing:meter:water:level": {Symbol: "", Title: "Water level", Description: "Dedicated water level meter"},
  "hiveot:thing:sensor": {Symbol: "", Title: "Sensor", Description: "Generic sensor device"},
  "hiveot:thing:sensor:security": {Symbol: "", Title: "Security", Description: "Generic security sensor"},
  "hiveot:thing:meter:electric": {Symbol: "", Title: "", Description: ""},
  "hiveot:thing:actuator:light": {Symbol: "", Title: "Light", Description: "Smart LED or other light"},
  "hiveot:thing:actuator:motor": {Symbol: "", Title: "Motor", Description: "Motor driven actuator, such as garage door, blinds, tv lifts"},
  "hiveot:thing:appliance:fridge": {Symbol: "", Title: "Fridge", Description: "Refrigerator appliance"},
  "hiveot:thing:computer:memory": {Symbol: "", Title: "Memory", Description: "Stand-alone memory device such as eeprom or iButtons"},
  "hiveot:thing:computer:voipphone": {Symbol: "", Title: "VoIP Phone", Description: "Voice over IP phone"},
  "hiveot:thing:device:indicator": {Symbol: "", Title: "Indicator", Description: "Visual or audio indicator device"},
  "hiveot:thing:meter": {Symbol: "", Title: "Meter", Description: "General metering device"},
  "hiveot:thing:net:gateway": {Symbol: "", Title: "Gateway", Description: "Generic gateway device providing access to other devices"},
  "hiveot:thing:actuator:relay": {Symbol: "", Title: "Relay", Description: "Generic relay electrical switch"},
  "hiveot:thing:actuator:valve": {Symbol: "", Title: "Valve", Description: "Electric powered valve for fluids or gas"},
  "hiveot:thing:appliance": {Symbol: "", Title: "Appliance", Description: "Appliance to accomplish a particular task for occupant use"},
  "hiveot:thing:control:switch": {Symbol: "", Title: "Input switch", Description: "On or off switch input control"},
  "hiveot:thing:device:battery:monitor": {Symbol: "", Title: "Battery Monitor", Description: "Battery monitor and charge controller"},
  "hiveot:thing:media": {Symbol: "", Title: "A/V media", Description: "Generic device for audio/video media record or playback"},
  "hiveot:thing:sensor:scale": {Symbol: "", Title: "Scale", Description: "Electronic weigh scale"},
  "hiveot:thing:actuator:output": {Symbol: "", Title: "Output", Description: "General purpose electrical output signal"},
  "hiveot:thing:actuator:switch": {Symbol: "", Title: "Switch", Description: "An electric powered on/off switch for powering circuits"},
  "hiveot:thing:computer:pc": {Symbol: "", Title: "PC/Laptop", Description: "Personal computer/laptop"},
  "hiveot:thing:control:pool": {Symbol: "", Title: "Pool control", Description: "Device for controlling pool settings"},
  "hiveot:thing:control:toggle": {Symbol: "", Title: "Toggle switch", Description: "Toggle switch input control"},
  "hiveot:thing:meter:water:consumption": {Symbol: "", Title: "Water consumption meter", Description: "Water consumption meter"},
  "hiveot:thing:net:gateway:zigbee": {Symbol: "", Title: "Zigbee gateway", Description: "Gateway providing access to Zigbee devices"},
  "hiveot:thing:net:switch": {Symbol: "", Title: "Network switch", Description: "Network switch to connect computer devices to the network"},
  "hiveot:thing:control:climate": {Symbol: "", Title: "Climate control", Description: "Device for controlling climate of a space"},
  "hiveot:thing:control:thermostat": {Symbol: "", Title: "Thermostat", Description: "Thermostat HVAC control"},
  "hiveot:thing:media:tv": {Symbol: "", Title: "TV", Description: "Network connected television"},
  "hiveot:thing:meter:electric:energy": {Symbol: "", Title: "Electric energy", Description: "Electrical energy meter"},
  "hiveot:thing:meter:wind": {Symbol: "", Title: "Wind", Description: "Dedicated wind meter"},
  "hiveot:thing:net:gateway:onewire": {Symbol: "", Title: "1-Wire gateway", Description: "Gateway providing access to 1-wire devices"},
  "hiveot:thing:sensor:security:glass": {Symbol: "", Title: "Glass sensor", Description: "Dedicated sensor for detecting breaking of glass"},
}


// type: UnitClasses
// version: 0.1
// generated: 21 Feb 25 21:45 PST
// source: github.com/hiveot/hub/api/vocab/ht-unit-classes.yaml
// namespace: hiveot
export const UnitAmpere = "hiveot:unit:ampere";
export const UnitCandela = "hiveot:unit:candela";
export const UnitCelcius = "hiveot:unit:celcius";
export const UnitCount = "hiveot:unit:count";
export const UnitDegree = "hiveot:unit:degree";
export const UnitFahrenheit = "hiveot:unit:fahrenheit";
export const UnitFoot = "hiveot:unit:foot";
export const UnitGallon = "hiveot:unit:gallon";
export const UnitKelvin = "hiveot:unit:kelvin";
export const UnitKilogram = "hiveot:unit:kilogram";
export const UnitKilometerPerHour = "hiveot:unit:kph";
export const UnitKilowattHour = "hiveot:unit:kilowatthour";
export const UnitLiter = "hiveot:unit:liter";
export const UnitLumen = "hiveot:unit:lumen";
export const UnitLux = "hiveot:unit:lux";
export const UnitMercury = "hiveot:unit:mercury";
export const UnitMeter = "hiveot:unit:meter";
export const UnitMeterPerSecond = "hiveot:unit:meterspersecond";
export const UnitMilesPerHour = "hiveot:unit:milesperhour";
export const UnitMilliSecond = "hiveot:unit:millisecond";
export const UnitMillibar = "hiveot:unit:millibar";
export const UnitMole = "hiveot:unit:mole";
export const UnitPSI = "hiveot:unit:psi";
export const UnitPascal = "hiveot:unit:pascal";
export const UnitPercent = "hiveot:unit:percent";
export const UnitPound = "hiveot:unit:pound";
export const UnitPpm = "hiveot:unit:ppm";
export const UnitRadian = "hiveot:unit:radian";
export const UnitSecond = "hiveot:unit:second";
export const UnitVolt = "hiveot:unit:volt";
export const UnitWatt = "hiveot:unit:watt";
// end of UnitClasses

// UnitClassesMap maps @type to symbol, title and description
export const UnitClassesMap = {
  "hiveot:unit:pascal": {Symbol: "Pa", Title: "Pascal", Description: "SI unit of pressure. Equal to 1 newton of force applied over 1 square meter."},
  "hiveot:unit:candela": {Symbol: "cd", Title: "Candela", Description: "SI unit of luminous intensity in a given direction. Roughly the same brightness as the common candle."},
  "hiveot:unit:gallon": {Symbol: "gl", Title: "Gallon", Description: "Unit of volume. 1 Imperial gallon is 4.54609 liters. 1 US liquid gallon is 3.78541 liters. 1 US dry gallon is 4.405 liters. "},
  "hiveot:unit:kelvin": {Symbol: "K", Title: "Kelvin", Description: "SI unit of thermodynamic temperature. 0 K represents absolute zero, the absence of all heat. 0 C equals +273.15K"},
  "hiveot:unit:milesperhour": {Symbol: "mph", Title: "Miles per hour", Description: "Speed in miles per hour"},
  "hiveot:unit:kilogram": {Symbol: "kg", Title: "Kilogram", Description: ""},
  "hiveot:unit:meter": {Symbol: "m", Title: "Meter", Description: "Distance in meters. 1m=c/299792458"},
  "hiveot:unit:mercury": {Symbol: "Hg", Title: "Mercury", Description: "Unit of atmospheric pressure in the United States. 1 Hg equals 33.8639 mbar."},
  "hiveot:unit:pound": {Symbol: "lbs", Title: "Pound", Description: "Imperial unit of weight. Equivalent to 0.453592 Kg. 1 Kg is 2.205 lbs"},
  "hiveot:unit:volt": {Symbol: "V", Title: "Volt", Description: "SI unit of electric potential; Energy consumption of 1 joule per electric charge of one coulomb"},
  "hiveot:unit:radian": {Symbol: "", Title: "Radian", Description: "Angle in 0-2pi"},
  "hiveot:unit:second": {Symbol: "s", Title: "Second", Description: "SI unit of time based on caesium frequency"},
  "hiveot:unit:ampere": {Symbol: "A", Title: "Ampere", Description: "Electrical current in Amperes based on the elementary charge flow per second"},
  "hiveot:unit:kilowatthour": {Symbol: "kWh", Title: "Kilowatt-hour", Description: "non-SI unit of energy equivalent to 3.6 megajoules."},
  "hiveot:unit:lumen": {Symbol: "lm", Title: "Lumen", Description: "SI unit luminous flux. Measure of perceived power of visible light. 1lm = 1 cd steradian"},
  "hiveot:unit:ppm": {Symbol: "ppm", Title: "PPM", Description: "Parts per million"},
  "hiveot:unit:celcius": {Symbol: "°C", Title: "Celcius", Description: "Temperature in Celcius"},
  "hiveot:unit:fahrenheit": {Symbol: "F", Title: "Fahrenheit", Description: "Temperature in Fahrenheit"},
  "hiveot:unit:degree": {Symbol: "degree", Title: "Degree", Description: "Angle in 0-360 degrees"},
  "hiveot:unit:millibar": {Symbol: "mbar", Title: "millibar", Description: "Metric unit of pressure. 1/1000th of a bar. Equal to 100 pascals. Amount of force it takes to move an object weighing a gram, one centimeter in one second."},
  "hiveot:unit:psi": {Symbol: "PSI", Title: "PSI", Description: "Unit of pressure. Pounds of force per square inch. 1PSI equals 6984 Pascals."},
  "hiveot:unit:kph": {Symbol: "kph", Title: "Km per hour", Description: "Speed in kilometers per hour"},
  "hiveot:unit:meterspersecond": {Symbol: "m/s", Title: "Meters per second", Description: "SI unit of speed in meters per second"},
  "hiveot:unit:millisecond": {Symbol: "ms", Title: "millisecond", Description: "Unit of time in milli-seconds. Equal to 1/1000 of a second."},
  "hiveot:unit:mole": {Symbol: "mol", Title: "Mole", Description: "SI unit of measurement for amount of substance. Eg, molecules."},
  "hiveot:unit:percent": {Symbol: "%", Title: "Percent", Description: "Fractions of 100"},
  "hiveot:unit:watt": {Symbol: "W", Title: "Watt", Description: "SI unit of power. Equal to 1 joule per second; or work performed when a current of 1 ampere flows across an electric potential of one volt."},
  "hiveot:unit:count": {Symbol: "(N)", Title: "Count", Description: ""},
  "hiveot:unit:foot": {Symbol: "ft", Title: "Foot", Description: "Imperial unit of distance. 1 foot equals 0.3048 meters"},
  "hiveot:unit:liter": {Symbol: "l", Title: "Liter", Description: "SI unit of volume equivalent to 1 cubic decimeter."},
  "hiveot:unit:lux": {Symbol: "lx", Title: "Lux", Description: "SI unit illuminance. Equal to 1 lumen per square meter."},
}


// type: ActionClasses
// version: 0.1
// generated: 21 Feb 25 21:45 PST
// source: github.com/hiveot/hub/api/vocab/ht-action-classes.yaml
// namespace: hiveot
export const ActionDimmer = "hiveot:action:dimmer";
export const ActionDimmerDecrement = "hiveot:action:dimmer:decrement";
export const ActionDimmerIncrement = "hiveot:action:dimmer:increment";
export const ActionDimmerSet = "hiveot:action:dimmer:set";
export const ActionMedia = "hiveot:action:media";
export const ActionMediaMute = "hiveot:action:media:mute";
export const ActionMediaNext = "hiveot:action:media:next";
export const ActionMediaPause = "hiveot:action:media:pause";
export const ActionMediaPlay = "hiveot:action:media:play";
export const ActionMediaPrevious = "hiveot:action:media:previous";
export const ActionMediaUnmute = "hiveot:action:media:unmute";
export const ActionMediaVolume = "hiveot:action:media:volume";
export const ActionMediaVolumeDecrease = "hiveot:action:media:volume:decrease";
export const ActionMediaVolumeIncrease = "hiveot:action:media:volume:increase";
export const ActionSwitch = "hiveot:action:switch";
export const ActionSwitchOnOff = "hiveot:action:switch:onoff";
export const ActionSwitchToggle = "hiveot:action:switch:toggle";
export const ActionThingDisable = "hiveot:action:thing:disable";
export const ActionThingEnable = "hiveot:action:thing:enable";
export const ActionThingStart = "hiveot:action:thing:start";
export const ActionThingStop = "hiveot:action:thing:stop";
export const ActionValveClose = "hiveot:action:valve:close";
export const ActionValveOpen = "hiveot:action:valve:open";
// end of ActionClasses

// ActionClassesMap maps @type to symbol, title and description
export const ActionClassesMap = {
  "hiveot:action:valve:close": {Symbol: "", Title: "Close valve", Description: "Action to close the valve"},
  "hiveot:action:media:next": {Symbol: "", Title: "Next", Description: "Next track or station"},
  "hiveot:action:media:play": {Symbol: "", Title: "Play", Description: "Start or continue playback"},
  "hiveot:action:switch:onoff": {Symbol: "", Title: "Set On/Off switch", Description: "Action to set the switch on/off state"},
  "hiveot:action:thing:enable": {Symbol: "", Title: "Enable", Description: "Action to enable a thing"},
  "hiveot:action:media:volume": {Symbol: "", Title: "Volume", Description: "Set volume level"},
  "hiveot:action:dimmer": {Symbol: "", Title: "Dimmer", Description: "General dimmer action"},
  "hiveot:action:dimmer:set": {Symbol: "", Title: "Set dimmer", Description: "Action to set the dimmer value"},
  "hiveot:action:thing:disable": {Symbol: "", Title: "Disable", Description: "Action to disable a thing"},
  "hiveot:action:media": {Symbol: "", Title: "Media control", Description: "Commands to control media recording and playback"},
  "hiveot:action:media:mute": {Symbol: "", Title: "Mute", Description: "Mute audio"},
  "hiveot:action:media:pause": {Symbol: "", Title: "Pause", Description: "Pause playback"},
  "hiveot:action:media:previous": {Symbol: "", Title: "Previous", Description: "Previous track or station"},
  "hiveot:action:thing:stop": {Symbol: "", Title: "Stop", Description: "Stop a running task"},
  "hiveot:action:media:volume:increase": {Symbol: "", Title: "Increase volume", Description: "Increase volume"},
  "hiveot:action:switch": {Symbol: "", Title: "Switch", Description: "General switch action"},
  "hiveot:action:switch:toggle": {Symbol: "", Title: "Toggle switch", Description: "Action to toggle the switch"},
  "hiveot:action:thing:start": {Symbol: "", Title: "Start", Description: "Start running a task"},
  "hiveot:action:valve:open": {Symbol: "", Title: "Open valve", Description: "Action to open the valve"},
  "hiveot:action:media:unmute": {Symbol: "", Title: "Unmute", Description: "Unmute audio"},
  "hiveot:action:media:volume:decrease": {Symbol: "", Title: "Decrease volume", Description: "Decrease volume"},
  "hiveot:action:dimmer:decrement": {Symbol: "", Title: "Lower dimmer", Description: ""},
  "hiveot:action:dimmer:increment": {Symbol: "", Title: "Increase dimmer", Description: ""},
}


// type: PropertyClasses
// version: 0.1
// generated: 21 Feb 25 21:45 PST
// source: github.com/hiveot/hub/api/vocab/ht-property-classes.yaml
// namespace: hiveot
export const PropAlarmMotion = "hiveot:prop:alarm:motion";
export const PropAlarmStatus = "hiveot:prop:alarm:status";
export const PropDevice = "hiveot:prop:device";
export const PropDeviceBattery = "hiveot:prop:device:battery";
export const PropDeviceDescription = "hiveot:prop:device:description";
export const PropDeviceEnabledDisabled = "hiveot:prop:device:enabled-disabled";
export const PropDeviceFirmwareVersion = "hiveot:prop:device:firmwareversion";
export const PropDeviceHardwareVersion = "hiveot:prop:device:hardwareversion";
export const PropDeviceMake = "hiveot:prop:device:make";
export const PropDeviceModel = "hiveot:prop:device:model";
export const PropDevicePollinterval = "hiveot:prop:device:pollinterval";
export const PropDeviceSoftwareVersion = "hiveot:prop:device:softwareversion";
export const PropDeviceStatus = "hiveot:prop:device:status";
export const PropDeviceTitle = "hiveot:prop:device:title";
export const PropElectric = "hiveot:prop:electric";
export const PropElectricCurrent = "hiveot:prop:electric:current";
export const PropElectricEnergy = "hiveot:prop:electric:energy";
export const PropElectricOverload = "hiveot:prop:electric:overload";
export const PropElectricPower = "hiveot:prop:electric:poer";
export const PropElectricVoltage = "hiveot:prop:electric:voltage";
export const PropEnv = "hiveot:prop:env";
export const PropEnvAcceleration = "hiveot:prop:env:acceleration";
export const PropEnvAirquality = "hiveot:prop:env:airquality";
export const PropEnvBarometer = "hiveot:prop:env:barometer";
export const PropEnvCO = "hiveot:prop:env:co";
export const PropEnvCO2 = "hiveot:prop:env:co2";
export const PropEnvCpuload = "hiveot:prop:env:cpuload";
export const PropEnvDewpoint = "hiveot:prop:env:dewpoint";
export const PropEnvFuelFlowrate = "hiveot:prop:env:fuel:flowrate";
export const PropEnvFuelLevel = "hiveot:prop:env:fuel:level";
export const PropEnvHumidex = "hiveot:prop:env:humidex";
export const PropEnvHumidity = "hiveot:prop:env:humidity";
export const PropEnvLuminance = "hiveot:prop:env:luminance";
export const PropEnvPressure = "hiveot:prop:env:pressure";
export const PropEnvTemperature = "hiveot:prop:env:temperature";
export const PropEnvTimezone = "hiveot:prop:env:timezone";
export const PropEnvUV = "hiveot:prop:env:uv";
export const PropEnvVibration = "hiveot:prop:env:vibration";
export const PropEnvVolume = "hiveot:prop:env:volume";
export const PropEnvWaterFlowrate = "hiveot:prop:env:water:flowrate";
export const PropEnvWaterLevel = "hiveot:prop:env:water:level";
export const PropEnvWindHeading = "hiveot:prop:env:wind:heading";
export const PropEnvWindSpeed = "hiveot:prop:env:wind:speed";
export const PropLocation = "hiveot:prop:location";
export const PropLocationCity = "hiveot:prop:location:city";
export const PropLocationLatitude = "hiveot:prop:location:latitude";
export const PropLocationLongitude = "hiveot:prop:location:longitude";
export const PropLocationName = "hiveot:prop:location:name";
export const PropLocationStreet = "hiveot:prop:location:street";
export const PropLocationZipcode = "hiveot:prop:location:zipcode";
export const PropMedia = "hiveot:prop:media";
export const PropMediaMuted = "hiveot:prop:media:muted";
export const PropMediaPaused = "hiveot:prop:media:paused";
export const PropMediaPlaying = "hiveot:prop:media:playing";
export const PropMediaStation = "hiveot:prop:media:station";
export const PropMediaTrack = "hiveot:prop:media:track";
export const PropMediaVolume = "hiveot:prop:media:volume";
export const PropNet = "hiveot:prop:net";
export const PropNetAddress = "hiveot:prop:net:address";
export const PropNetConnection = "hiveot:prop:net:connection";
export const PropNetDomainname = "hiveot:prop:net:domainname";
export const PropNetGateway = "hiveot:prop:net:gateway";
export const PropNetHostname = "hiveot:prop:net:hostname";
export const PropNetIP4 = "hiveot:prop:net:ip4";
export const PropNetIP6 = "hiveot:prop:net:ip6";
export const PropNetLatency = "hiveot:prop:net:latency";
export const PropNetMAC = "hiveot:prop:net:mac";
export const PropNetMask = "hiveot:prop:net:mask";
export const PropNetPort = "hiveot:prop:net:port";
export const PropNetSignalstrength = "hiveot:prop:net:signalstrength";
export const PropNetSubnet = "hiveot:prop:net:subnet";
export const PropStatusOnOff = "hiveot:prop:status:onoff";
export const PropStatusOpenClosed = "hiveot:prop:status:openclosed";
export const PropStatusStartedStopped = "hiveot:prop:status:started-stopped";
export const PropStatusYesNo = "hiveot:prop:status:yes-no";
export const PropSwitch = "hiveot:prop:switch";
export const PropSwitchDimmer = "hiveot:prop:switch:dimmer";
export const PropSwitchLight = "hiveot:prop:switch:light";
export const PropSwitchLocked = "hiveot:prop:switch:locked";
export const PropSwitchOnOff = "hiveot:prop:switch:onoff";
// end of PropertyClasses

// PropertyClassesMap maps @type to symbol, title and description
export const PropertyClassesMap = {
  "hiveot:prop:env:wind:heading": {Symbol: "", Title: "Wind heading", Description: ""},
  "hiveot:prop:env:volume": {Symbol: "", Title: "Volume", Description: ""},
  "hiveot:prop:device:model": {Symbol: "", Title: "Model", Description: "Device model"},
  "hiveot:prop:electric:voltage": {Symbol: "", Title: "Voltage", Description: "Electrical voltage potential"},
  "hiveot:prop:electric:overload": {Symbol: "", Title: "Overload protection", Description: "Cut load on overload"},
  "hiveot:prop:env:humidity": {Symbol: "", Title: "Humidity", Description: ""},
  "hiveot:prop:location:name": {Symbol: "", Title: "Location name", Description: "Name of the location"},
  "hiveot:prop:net:connection": {Symbol: "", Title: "Connection", Description: "Connection status, connected, connecting, retrying, disconnected,..."},
  "hiveot:prop:status:onoff": {Symbol: "", Title: "On/off status", Description: ""},
  "hiveot:prop:alarm:motion": {Symbol: "", Title: "Motion", Description: "Motion detected"},
  "hiveot:prop:device:status": {Symbol: "", Title: "Status", Description: "Device status; alive, awake, dead, sleeping"},
  "hiveot:prop:env:co": {Symbol: "", Title: "Carbon monoxide level", Description: "Carbon monoxide level"},
  "hiveot:prop:env:pressure": {Symbol: "", Title: "Pressure", Description: ""},
  "hiveot:prop:location:latitude": {Symbol: "", Title: "Latitude", Description: "Latitude geographic coordinate"},
  "hiveot:prop:media:muted": {Symbol: "", Title: "Muted", Description: "Audio is muted"},
  "hiveot:prop:switch:light": {Symbol: "", Title: "Light switch", Description: ""},
  "hiveot:prop:device:pollinterval": {Symbol: "", Title: "Polling interval", Description: "Interval to poll for updates"},
  "hiveot:prop:env:wind:speed": {Symbol: "", Title: "Wind speed", Description: ""},
  "hiveot:prop:media:track": {Symbol: "", Title: "Track", Description: "Selected A/V track"},
  "hiveot:prop:net:domainname": {Symbol: "", Title: "Domain name", Description: "Domainname of the client"},
  "hiveot:prop:net:gateway": {Symbol: "", Title: "Gateway", Description: "Network gateway address"},
  "hiveot:prop:switch:locked": {Symbol: "", Title: "Lock", Description: "Electric lock status"},
  "hiveot:prop:electric:current": {Symbol: "", Title: "Current", Description: "Electrical current"},
  "hiveot:prop:env:vibration": {Symbol: "", Title: "Vibration", Description: ""},
  "hiveot:prop:status:yes-no": {Symbol: "", Title: "Yes/No", Description: "Status with yes or no value"},
  "hiveot:prop:device:enabled-disabled": {Symbol: "", Title: "Enabled/Disabled", Description: "Enabled or disabled state"},
  "hiveot:prop:env:humidex": {Symbol: "", Title: "Humidex", Description: ""},
  "hiveot:prop:env:luminance": {Symbol: "", Title: "Luminance", Description: ""},
  "hiveot:prop:env:uv": {Symbol: "", Title: "UV", Description: ""},
  "hiveot:prop:net:latency": {Symbol: "", Title: "Network latency", Description: "Delay between hub and client"},
  "hiveot:prop:net:mask": {Symbol: "", Title: "Netmask", Description: "Network mask. Example: 255.255.255.0 or 24/8"},
  "hiveot:prop:electric:poer": {Symbol: "", Title: "Power", Description: "Electrical power being consumed"},
  "hiveot:prop:env:dewpoint": {Symbol: "", Title: "Dew point", Description: "Dew point temperature"},
  "hiveot:prop:env:timezone": {Symbol: "", Title: "Timezone", Description: ""},
  "hiveot:prop:switch:onoff": {Symbol: "", Title: "On/Off switch", Description: ""},
  "hiveot:prop:device:firmwareversion": {Symbol: "", Title: "Firmware version", Description: ""},
  "hiveot:prop:device:softwareversion": {Symbol: "", Title: "Software version", Description: ""},
  "hiveot:prop:electric:energy": {Symbol: "", Title: "Energy", Description: "Electrical energy consumed"},
  "hiveot:prop:env:cpuload": {Symbol: "", Title: "CPU load level", Description: "Device CPU load level"},
  "hiveot:prop:media:volume": {Symbol: "", Title: "Volume", Description: "Media volume setting"},
  "hiveot:prop:net": {Symbol: "", Title: "Network properties", Description: "General network properties"},
  "hiveot:prop:net:ip4": {Symbol: "", Title: "IP4 address", Description: "Device IP4 address"},
  "hiveot:prop:net:mac": {Symbol: "", Title: "MAC", Description: "Hardware MAC address"},
  "hiveot:prop:net:subnet": {Symbol: "", Title: "Subnet", Description: "Network subnet address. Example: 192.168.0.0"},
  "hiveot:prop:device:title": {Symbol: "", Title: "Title", Description: "Device friendly title"},
  "hiveot:prop:env:co2": {Symbol: "", Title: "Carbon dioxide level", Description: "Carbon dioxide level"},
  "hiveot:prop:location:city": {Symbol: "", Title: "City", Description: "City name"},
  "hiveot:prop:net:port": {Symbol: "", Title: "Port", Description: "Network port"},
  "hiveot:prop:net:signalstrength": {Symbol: "", Title: "Signal strength", Description: "Wireless signal strength"},
  "hiveot:prop:alarm:status": {Symbol: "", Title: "Alarm state", Description: "Current alarm status"},
  "hiveot:prop:env:water:flowrate": {Symbol: "", Title: "Water flow rate", Description: ""},
  "hiveot:prop:media:playing": {Symbol: "", Title: "Playing", Description: "Media is playing"},
  "hiveot:prop:status:openclosed": {Symbol: "", Title: "Open/Closed status", Description: ""},
  "hiveot:prop:device:description": {Symbol: "", Title: "Description", Description: "Device product description"},
  "hiveot:prop:device:hardwareversion": {Symbol: "", Title: "Hardware version", Description: ""},
  "hiveot:prop:electric": {Symbol: "", Title: "Electrical properties", Description: "General group of electrical properties"},
  "hiveot:prop:location": {Symbol: "", Title: "Location", Description: "General location information"},
  "hiveot:prop:net:address": {Symbol: "", Title: "Address", Description: "Network address"},
  "hiveot:prop:net:hostname": {Symbol: "", Title: "Hostname", Description: "Hostname of the client"},
  "hiveot:prop:media:station": {Symbol: "", Title: "Station", Description: "Selected radio station"},
  "hiveot:prop:status:started-stopped": {Symbol: "", Title: "Started/Stopped", Description: "Started or stopped status"},
  "hiveot:prop:device": {Symbol: "", Title: "Device attributes", Description: "Attributes describing a device"},
  "hiveot:prop:device:make": {Symbol: "", Title: "Make", Description: "Device manufacturer"},
  "hiveot:prop:env:acceleration": {Symbol: "", Title: "Acceleration", Description: ""},
  "hiveot:prop:env:barometer": {Symbol: "", Title: "Atmospheric pressure", Description: "Barometric pressure of the atmosphere"},
  "hiveot:prop:env:fuel:flowrate": {Symbol: "", Title: "Fuel flow rate", Description: ""},
  "hiveot:prop:env:water:level": {Symbol: "", Title: "Water level", Description: ""},
  "hiveot:prop:device:battery": {Symbol: "", Title: "Battery level", Description: "Device battery level"},
  "hiveot:prop:env:airquality": {Symbol: "", Title: "Air quality", Description: "Air quality level"},
  "hiveot:prop:media:paused": {Symbol: "", Title: "Paused", Description: "Media is paused"},
  "hiveot:prop:net:ip6": {Symbol: "", Title: "IP6 address", Description: "Device IP6 address"},
  "hiveot:prop:media": {Symbol: "", Title: "Media commands", Description: "Control of media equipment"},
  "hiveot:prop:switch": {Symbol: "", Title: "Switch status", Description: ""},
  "hiveot:prop:env": {Symbol: "", Title: "Environmental property", Description: "Property of environmental sensor"},
  "hiveot:prop:env:fuel:level": {Symbol: "", Title: "Fuel level", Description: ""},
  "hiveot:prop:env:temperature": {Symbol: "", Title: "Temperature", Description: ""},
  "hiveot:prop:location:street": {Symbol: "", Title: "Street", Description: "Street address"},
  "hiveot:prop:location:longitude": {Symbol: "", Title: "Longitude", Description: "Longitude geographic coordinate"},
  "hiveot:prop:location:zipcode": {Symbol: "", Title: "Zip code", Description: "Location ZIP code"},
  "hiveot:prop:switch:dimmer": {Symbol: "", Title: "Dimmer value", Description: ""},
}
