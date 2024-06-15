// Package vocab with HiveOT and WoT vocabulary names for TD Things, properties, events and actions
// DO NOT EDIT. This file is generated and changes will be overwritten
package vocab

// type: MessageTypes
// version: 0.1
// generated: 09 Jun 24 09:35 PDT
// source: github.com/hiveot/hub/api/vocab/ht-constants.yaml
// description: Message types used throughout the Hub and its clients
const (
	ActionTypeProperties    = "$properties"
	EventTypeDeliveryUpdate = "$delivery"
	EventTypeProperties     = "$properties"
	EventTypeTD             = "$td"
	MessageTypeAction       = "action"
	MessageTypeEvent        = "event"
)

// end of MessageTypes

// type: DigiTwinRestPaths
// version: 0.1
// generated: 09 Jun 24 09:35 PDT
// source: github.com/hiveot/hub/api/vocab/ht-constants.yaml
// description: DigiTwin rest API paths for use by agents and consumers
const (
	ConnectSSEPath      = "/sse"
	GetEventsPath       = "/things/{thingID}/events"
	GetThingsPath       = "/things"
	PostActionPath      = "/things/{thingID}/action/{key}"
	PostEventPath       = "/things/{thingID}/event/{key}"
	PostLoginPath       = "/authn/login"
	PostLogoutPath      = "/authn/logout"
	PostRefreshPath     = "/authn/refresh"
	PostSubscribePath   = "/subscribe/{thingID}/{key}"
	PostUnsubscribePath = "/unsubscribe/{thingID}/{key}"
)

// end of DigiTwinRestPaths

// type: WoTVocab
// version: 0.1
// generated: 09 Jun 24 09:35 PDT
// source: github.com/hiveot/hub/api/vocab/wot-vocab.yaml
// description: WoT vocabulary definition. See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
const (
	WoTAPIKeySecurityScheme = "APIKeySecurityScheme"
	WoTActions              = "actions"
	WoTAnyURI               = "https://www.w3.org/2019/wot/thing/v1"
	WoTAtContext            = "@context"
	WoTAtType               = "@type"
	WoTBasicSecurityScheme  = "BasicSecurityScheme"
	WoTBearerSecurityScheme = "BearerSecurityScheme"
	WoTConst                = "const"
	WoTCreated              = "created"
	WoTDataType             = "type"
	WoTDataTypeAnyURI       = "anyURI"
	WoTDataTypeArray        = "array"
	WoTDataTypeBool         = "boolean"
	WoTDataTypeDateTime     = "dateTime"
	WoTDataTypeInteger      = "integer"
	WoTDataTypeNone         = ""
	WoTDataTypeNumber       = "number"
	WoTDataTypeObject       = "object"
	WoTDataTypeString       = "string"
	WoTDataTypeUnsignedInt  = "unsignedInt"
	WoTDescription          = "description"
	WoTDescriptions         = "descriptions"
	WoTDigestSecurityScheme = "DigestSecurityScheme"
	WoTEnum                 = "enum"
	WoTEvents               = "events"
	WoTFormat               = "format"
	WoTForms                = "forms"
	WoTHref                 = "href"
	WoTID                   = "id"
	WoTInput                = "input"
	WoTLinks                = "links"
	WoTMaxItems             = "maxItems"
	WoTMaxLength            = "maxLength"
	WoTMaximum              = "maximum"
	WoTMinItems             = "minItems"
	WoTMinLength            = "minLength"
	WoTMinimum              = "minimum"
	WoTModified             = "modified"
	WoTNoSecurityScheme     = "NoSecurityScheme"
	WoTOAuth2SecurityScheme = "OAuth2SecurityScheme"
	WoTOperation            = "op"
	WoTOutput               = "output"
	WoTPSKSecurityScheme    = "PSKSecurityScheme"
	WoTProperties           = "properties"
	WoTReadOnly             = "readOnly"
	WoTRequired             = "required"
	WoTSecurity             = "security"
	WoTSupport              = "support"
	WoTTitle                = "title"
	WoTTitles               = "titles"
	WoTVersion              = "version"
)

// end of WoTVocab

// type: ThingClasses
// version: 0.1
// generated: 09 Jun 24 09:35 PDT
// source: github.com/hiveot/hub/api/vocab/ht-thing-classes.yaml
// namespace: ht
const (
	ThingActuator                 = "ht:thing:actuator"
	ThingActuatorAlarm            = "ht:thing:actuator:alarm"
	ThingActuatorBeacon           = "ht:thing:actuator:beacon"
	ThingActuatorDimmer           = "ht:thing:actuator:dimmer"
	ThingActuatorLight            = "ht:thing:actuator:light"
	ThingActuatorLock             = "ht:thing:actuator:lock"
	ThingActuatorMotor            = "ht:thing:actuator:motor"
	ThingActuatorOutput           = "ht:thing:actuator:output"
	ThingActuatorRanged           = "ht:thing:actuator:ranged"
	ThingActuatorRelay            = "ht:thing:actuator:relay"
	ThingActuatorSwitch           = "ht:thing:actuator:switch"
	ThingActuatorValve            = "ht:thing:actuator:valve"
	ThingActuatorValveFuel        = "ht:thing:actuator:valve:fuel"
	ThingActuatorValveWater       = "ht:thing:actuator:valve:water"
	ThingAppliance                = "ht:thing:appliance"
	ThingApplianceDishwasher      = "ht:thing:appliance:dishwasher"
	ThingApplianceDryer           = "ht:thing:appliance:dryer"
	ThingApplianceFreezer         = "ht:thing:appliance:freezer"
	ThingApplianceFridge          = "ht:thing:appliance:fridge"
	ThingApplianceWasher          = "ht:thing:appliance:washer"
	ThingComputer                 = "ht:thing:computer"
	ThingComputerCellphone        = "ht:thing:computer:cellphone"
	ThingComputerEmbedded         = "ht:thing:computer:embedded"
	ThingComputerMemory           = "ht:thing:computer:memory"
	ThingComputerPC               = "ht:thing:computer:pc"
	ThingComputerPotsPhone        = "ht:thing:computer:potsphone"
	ThingComputerSatPhone         = "ht:thing:computer:satphone"
	ThingComputerTablet           = "ht:thing:computer:tablet"
	ThingComputerVoipPhone        = "ht:thing:computer:voipphone"
	ThingControl                  = "ht:thing:control"
	ThingControlClimate           = "ht:thing:control:climate"
	ThingControlDimmer            = "ht:thing:control:dimmer"
	ThingControlIrrigation        = "ht:thing:control:irrigation"
	ThingControlJoystick          = "ht:thing:control:joystick"
	ThingControlKeypad            = "ht:thing:control:keypad"
	ThingControlPool              = "ht:thing:control:pool"
	ThingControlPushbutton        = "ht:thing:control:pushbutton"
	ThingControlSwitch            = "ht:thing:control:switch"
	ThingControlThermostat        = "ht:thing:control:thermostat"
	ThingControlToggle            = "ht:thing:control:toggle"
	ThingDevice                   = "ht:thing:device"
	ThingDeviceBatteryMonitor     = "ht:thing:device:battery:monitor"
	ThingDeviceIndicator          = "ht:thing:device:indicator"
	ThingDeviceTime               = "ht:thing:device:time"
	ThingMedia                    = "ht:thing:media"
	ThingMediaAmplifier           = "ht:thing:media:amplifier"
	ThingMediaCamera              = "ht:thing:media:camera"
	ThingMediaMicrophone          = "ht:thing:media:microphone"
	ThingMediaPlayer              = "ht:thing:media:player"
	ThingMediaRadio               = "ht:thing:media:radio"
	ThingMediaReceiver            = "ht:thing:media:receiver"
	ThingMediaSpeaker             = "ht:thing:media:speaker"
	ThingMediaTV                  = "ht:thing:media:tv"
	ThingMeter                    = "ht:thing:meter"
	ThingMeterElectric            = "ht:thing:meter:electric"
	ThingMeterElectricCurrent     = "ht:thing:meter:electric:current"
	ThingMeterElectricEnergy      = "ht:thing:meter:electric:energy"
	ThingMeterElectricPower       = "ht:thing:meter:electric:power"
	ThingMeterElectricVoltage     = "ht:thing:meter:electric:voltage"
	ThingMeterFuel                = "ht:thing:meter:fuel"
	ThingMeterFuelFlow            = "ht:thing:meter:fuel:flow"
	ThingMeterFuelLevel           = "ht:thing:meter:fuel:level"
	ThingMeterWater               = "ht:thing:meter:water"
	ThingMeterWaterConsumption    = "ht:thing:meter:water:consumption"
	ThingMeterWaterFlow           = "ht:thing:meter:water:flow"
	ThingMeterWaterLevel          = "ht:thing:meter:water:level"
	ThingMeterWind                = "ht:thing:meter:wind"
	ThingNet                      = "ht:thing:net"
	ThingNetBluetooth             = "ht:thing:net:bluetooth"
	ThingNetGateway               = "ht:thing:net:gateway"
	ThingNetGatewayCoap           = "ht:thing:net:gateway:coap"
	ThingNetGatewayInsteon        = "ht:thing:net:gateway:insteon"
	ThingNetGatewayOnewire        = "ht:thing:net:gateway:onewire"
	ThingNetGatewayZigbee         = "ht:thing:net:gateway:zigbee"
	ThingNetGatewayZwave          = "ht:thing:net:gateway:zwave"
	ThingNetLora                  = "ht:thing:net:lora"
	ThingNetLoraGateway           = "ht:thing:net:lora:gw"
	ThingNetLoraP2P               = "ht:thing:net:lora:p2p"
	ThingNetRouter                = "ht:thing:net:router"
	ThingNetSwitch                = "ht:thing:net:switch"
	ThingNetWifi                  = "ht:thing:net:wifi"
	ThingNetWifiAp                = "ht:thing:net:wifi:ap"
	ThingSensor                   = "ht:thing:sensor"
	ThingSensorEnvironment        = "ht:thing:sensor:environment"
	ThingSensorInput              = "ht:thing:sensor:input"
	ThingSensorMulti              = "ht:thing:sensor:multi"
	ThingSensorScale              = "ht:thing:sensor:scale"
	ThingSensorSecurity           = "ht:thing:sensor:security"
	ThingSensorSecurityDoorWindow = "ht:thing:sensor:security:doorwindow"
	ThingSensorSecurityGlass      = "ht:thing:sensor:security:glass"
	ThingSensorSecurityMotion     = "ht:thing:sensor:security:motion"
	ThingSensorSmoke              = "ht:thing:sensor:smoke"
	ThingSensorSound              = "ht:thing:sensor:sound"
	ThingSensorThermometer        = "ht:thing:sensor:thermometer"
	ThingSensorWaterLeak          = "ht:thing:sensor:water:leak"
	ThingService                  = "ht:thing:service"
	ThingServiceAdapter           = "ht:thing:service:adapter"
	ThingServiceAuth              = "ht:thing:service:auth"
	ThingServiceAutomation        = "ht:thing:service:automation"
	ThingServiceDirectory         = "ht:thing:service:directory"
	ThingServiceHistory           = "ht:thing:service:history"
	ThingServiceImage             = "ht:thing:service:image"
	ThingServiceSTT               = "ht:thing:service:stt"
	ThingServiceStore             = "ht:thing:service:store"
	ThingServiceTTS               = "ht:thing:service:tts"
	ThingServiceTranslation       = "ht:thing:service:translation"
	ThingServiceWeather           = "ht:thing:service:weather"
	ThingServiceWeatherCurrent    = "ht:thing:service:weather:current"
	ThingServiceWeatherForecast   = "ht:thing:service:weather:forecast"
)

// end of ThingClasses

// ThingClassesMap maps @type to symbol, title and description
var ThingClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	ThingActuatorBeacon:           {Symbol: "", Title: "Beacon", Description: "Location beacon"},
	ThingControlIrrigation:        {Symbol: "", Title: "Irrigation control", Description: "Device for control of an irrigation system"},
	ThingDeviceBatteryMonitor:     {Symbol: "", Title: "Battery Monitor", Description: "Battery monitor and charge controller"},
	ThingActuatorOutput:           {Symbol: "", Title: "Output", Description: "General purpose electrical output signal"},
	ThingApplianceDishwasher:      {Symbol: "", Title: "Dishwasher", Description: "Dishwasher"},
	ThingDevice:                   {Symbol: "", Title: "Device", Description: "Device of unknown purpose"},
	ThingControlPushbutton:        {Symbol: "", Title: "Momentary switch", Description: "Momentary push button control input"},
	ThingMeterFuel:                {Symbol: "", Title: "Fuel metering device", Description: "General fuel metering device"},
	ThingComputerPotsPhone:        {Symbol: "", Title: "Land Line", Description: "Plain Old Telephone System, aka landline"},
	ThingServiceAuth:              {Symbol: "", Title: "Authentication service", Description: ""},
	ThingMediaCamera:              {Symbol: "", Title: "Camera", Description: "Video camera"},
	ThingMediaRadio:               {Symbol: "", Title: "Radio", Description: "AM or FM radio receiver"},
	ThingMeterElectricPower:       {Symbol: "", Title: "Electrical Power", Description: "Electrical power meter"},
	ThingMediaMicrophone:          {Symbol: "", Title: "Microphone", Description: "Microphone for capturing audio"},
	ThingSensorMulti:              {Symbol: "", Title: "Multi sensor", Description: "Sense multiple inputs"},
	ThingSensorScale:              {Symbol: "", Title: "Scale", Description: "Electronic weigh scale"},
	ThingComputerEmbedded:         {Symbol: "", Title: "Embedded System", Description: "Embedded computing device"},
	ThingControlToggle:            {Symbol: "", Title: "Toggle switch", Description: "Toggle switch input control"},
	ThingMediaPlayer:              {Symbol: "", Title: "Media player", Description: "CD/DVD/Blueray/USB player of recorded media"},
	ThingMeterElectricCurrent:     {Symbol: "", Title: "Electric current", Description: "Electrical current meter"},
	ThingMeterWater:               {Symbol: "", Title: "Water metering device", Description: "General water metering device"},
	ThingNetRouter:                {Symbol: "", Title: "Network router", Description: "IP ThingNetwork router providing access to other IP networks"},
	ThingSensorSecurity:           {Symbol: "", Title: "Security", Description: "Generic security sensor"},
	ThingControlKeypad:            {Symbol: "", Title: "Keypad", Description: "Multi-key pad for command input"},
	ThingMeterWaterLevel:          {Symbol: "", Title: "Water level", Description: "Dedicated water level meter"},
	ThingNetBluetooth:             {Symbol: "", Title: "Bluetooth", Description: "Bluetooth radio"},
	ThingSensorSecurityMotion:     {Symbol: "", Title: "Motion sensor", Description: "Dedicated security sensor detecting motion"},
	ThingServiceSTT:               {Symbol: "", Title: "Speech to text", Description: ""},
	ThingActuatorDimmer:           {Symbol: "", Title: "Dimmer", Description: "Light dimmer"},
	ThingMeter:                    {Symbol: "", Title: "Meter", Description: "General metering device"},
	ThingSensorSmoke:              {Symbol: "", Title: "Smoke detector", Description: ""},
	ThingComputerSatPhone:         {Symbol: "", Title: "Satellite phone", Description: ""},
	ThingMeterElectricEnergy:      {Symbol: "", Title: "Electric energy", Description: "Electrical energy meter"},
	ThingMeterElectricVoltage:     {Symbol: "", Title: "Voltage", Description: "Electrical voltage meter"},
	ThingMeterWind:                {Symbol: "", Title: "Wind", Description: "Dedicated wind meter"},
	ThingServiceWeather:           {Symbol: "", Title: "Weather service", Description: "General weather service"},
	ThingMedia:                    {Symbol: "", Title: "A/V media", Description: "Generic device for audio/video media record or playback"},
	ThingServiceHistory:           {Symbol: "", Title: "History service", Description: ""},
	ThingApplianceFreezer:         {Symbol: "", Title: "Freezer", Description: "Refrigerator freezer"},
	ThingComputer:                 {Symbol: "", Title: "Computing Device", Description: "General purpose computing device"},
	ThingServiceDirectory:         {Symbol: "", Title: "Directory service", Description: ""},
	ThingServiceWeatherCurrent:    {Symbol: "", Title: "Current weather", Description: ""},
	ThingActuatorValve:            {Symbol: "", Title: "Valve", Description: "Electric powered valve for fluids or gas"},
	ThingMeterWaterConsumption:    {Symbol: "", Title: "Water consumption meter", Description: "Water consumption meter"},
	ThingNetGatewayInsteon:        {Symbol: "", Title: "Insteon gateway", Description: "Gateway providing access to Insteon devices"},
	ThingNetWifiAp:                {Symbol: "", Title: "Wifi access point", Description: "Wireless access point for IP networks"},
	ThingServiceTTS:               {Symbol: "", Title: "Text to speech", Description: ""},
	ThingActuatorRelay:            {Symbol: "", Title: "Relay", Description: "Generic relay electrical switch"},
	ThingNetLora:                  {Symbol: "", Title: "LoRa network device", Description: "Generic Long Range network protocol device"},
	ThingComputerPC:               {Symbol: "", Title: "PC/Laptop", Description: "Personal computer/laptop"},
	ThingControl:                  {Symbol: "", Title: "Input controller", Description: "Generic input controller"},
	ThingDeviceTime:               {Symbol: "", Title: "Clock", Description: "Time tracking device such as clocks and time chips"},
	ThingMeterElectric:            {Symbol: "", Title: "", Description: ""},
	ThingNetGatewayZwave:          {Symbol: "", Title: "ZWave gateway", Description: "Gateway providing access to ZWave devices"},
	ThingServiceAdapter:           {Symbol: "", Title: "Protocol adapter", Description: "Protocol adapter/binding for integration with another protocol"},
	ThingServiceWeatherForecast:   {Symbol: "", Title: "Weather forecast", Description: ""},
	ThingActuatorMotor:            {Symbol: "", Title: "Motor", Description: "Motor driven actuator, such as garage door, blinds, tv lifts"},
	ThingComputerCellphone:        {Symbol: "", Title: "Cell Phone", Description: "Cellular phone"},
	ThingMediaReceiver:            {Symbol: "", Title: "Receiver", Description: "Audio/video receiver and player"},
	ThingComputerTablet:           {Symbol: "", Title: "Tablet", Description: "Tablet computer"},
	ThingMediaSpeaker:             {Symbol: "", Title: "Connected speakers", Description: "Network connected speakers"},
	ThingNetGatewayZigbee:         {Symbol: "", Title: "Zigbee gateway", Description: "Gateway providing access to Zigbee devices"},
	ThingServiceImage:             {Symbol: "", Title: "Image classification", Description: ""},
	ThingActuatorValveFuel:        {Symbol: "", Title: "Fuel valve", Description: "Electric powered fuel valve"},
	ThingMeterFuelLevel:           {Symbol: "", Title: "Fuel level", Description: "Dedicated fuel level metering device"},
	ThingNetGatewayOnewire:        {Symbol: "", Title: "1-Wire gateway", Description: "Gateway providing access to 1-wire devices"},
	ThingNetSwitch:                {Symbol: "", Title: "Network switch", Description: "Network switch to connect computer devices to the network"},
	ThingNetLoraGateway:           {Symbol: "", Title: "LoRaWAN gateway", Description: "Gateway providing access to LoRa devices"},
	ThingServiceTranslation:       {Symbol: "", Title: "Language translation service", Description: ""},
	ThingControlDimmer:            {Symbol: "", Title: "Dimmer", Description: "Light dimmer input device"},
	ThingControlThermostat:        {Symbol: "", Title: "Thermostat", Description: "Thermostat HVAC control"},
	ThingNetGatewayCoap:           {Symbol: "", Title: "CoAP gateway", Description: "Gateway providing access to CoAP devices"},
	ThingNetWifi:                  {Symbol: "", Title: "Wifi device", Description: "Generic wifi device"},
	ThingActuator:                 {Symbol: "", Title: "Actuator", Description: "Generic actuator"},
	ThingActuatorAlarm:            {Symbol: "", Title: "Alarm", Description: "Siren or light alarm"},
	ThingActuatorRanged:           {Symbol: "", Title: "Ranged actuator", Description: "Generic ranged actuator with a set point"},
	ThingApplianceWasher:          {Symbol: "", Title: "Washer", Description: "Clothing washer"},
	ThingControlClimate:           {Symbol: "", Title: "Climate control", Description: "Device for controlling climate of a space"},
	ThingSensor:                   {Symbol: "", Title: "Sensor", Description: "Generic sensor device"},
	ThingServiceStore:             {Symbol: "", Title: "Data storage", Description: ""},
	ThingControlPool:              {Symbol: "", Title: "Pool control", Description: "Device for controlling pool settings"},
	ThingSensorWaterLeak:          {Symbol: "", Title: "Water leak detector", Description: "Dedicated water leak detector"},
	ThingComputerMemory:           {Symbol: "", Title: "Memory", Description: "Stand-alone memory device such as eeprom or iButtons"},
	ThingDeviceIndicator:          {Symbol: "", Title: "Indicator", Description: "Visual or audio indicator device"},
	ThingMeterWaterFlow:           {Symbol: "", Title: "Water flow", Description: "Dedicated water flow-rate meter"},
	ThingNet:                      {Symbol: "", Title: "Network device", Description: "Generic network device"},
	ThingSensorInput:              {Symbol: "", Title: "Input sensor", Description: "General purpose electrical input sensor"},
	ThingActuatorValveWater:       {Symbol: "", Title: "Water valve", Description: "Electric powered water valve"},
	ThingApplianceFridge:          {Symbol: "", Title: "Fridge", Description: "Refrigerator appliance"},
	ThingSensorEnvironment:        {Symbol: "", Title: "Environmental sensor", Description: "Environmental sensor with one or more features such as temperature, humidity, etc"},
	ThingActuatorLock:             {Symbol: "", Title: "Lock", Description: "Electronic door lock"},
	ThingControlSwitch:            {Symbol: "", Title: "Input switch", Description: "On or off switch input control"},
	ThingSensorSecurityGlass:      {Symbol: "", Title: "Glass sensor", Description: "Dedicated sensor for detecting breaking of glass"},
	ThingSensorSecurityDoorWindow: {Symbol: "", Title: "Door/Window sensor", Description: "Dedicated door/window opening security sensor"},
	ThingSensorSound:              {Symbol: "", Title: "Sound detector", Description: ""},
	ThingSensorThermometer:        {Symbol: "", Title: "Thermometer", Description: "Environmental thermometer"},
	ThingControlJoystick:          {Symbol: "", Title: "Joystick", Description: "Flight control stick"},
	ThingMediaAmplifier:           {Symbol: "", Title: "Audio amplifier", Description: "Audio amplifier with volume controls"},
	ThingNetLoraP2P:               {Symbol: "", Title: "LoRa P2P", Description: "LoRa Peer-to-peer network device"},
	ThingActuatorLight:            {Symbol: "", Title: "Light", Description: "Smart LED or other light"},
	ThingActuatorSwitch:           {Symbol: "", Title: "Switch", Description: "An electric powered on/off switch for powering circuits"},
	ThingMediaTV:                  {Symbol: "", Title: "TV", Description: "Network connected television"},
	ThingService:                  {Symbol: "", Title: "Service", Description: "General service for processing data and offering features of interest"},
	ThingApplianceDryer:           {Symbol: "", Title: "Dryer", Description: "Clothing dryer"},
	ThingComputerVoipPhone:        {Symbol: "", Title: "VoIP Phone", Description: "Voice over IP phone"},
	ThingNetGateway:               {Symbol: "", Title: "Gateway", Description: "Generic gateway device providing access to other devices"},
	ThingAppliance:                {Symbol: "", Title: "Appliance", Description: "Appliance to accomplish a particular task for occupant use"},
	ThingMeterFuelFlow:            {Symbol: "", Title: "Fuel flow rate", Description: "Dedicated fuel flow rate metering device"},
	ThingServiceAutomation:        {Symbol: "", Title: "Automation service", Description: ""},
}

// type: UnitClasses
// version: 0.1
// generated: 09 Jun 24 09:35 PDT
// source: github.com/hiveot/hub/api/vocab/ht-unit-classes.yaml
// namespace: ht
const (
	UnitAmpere           = "ht:unit:ampere"
	UnitCandela          = "ht:unit:candela"
	UnitCelcius          = "ht:unit:celcius"
	UnitCount            = "ht:unit:count"
	UnitDegree           = "ht:unit:degree"
	UnitFahrenheit       = "ht:unit:fahrenheit"
	UnitFoot             = "ht:unit:foot"
	UnitGallon           = "ht:unit:gallon"
	UnitKelvin           = "ht:unit:kelvin"
	UnitKilogram         = "ht:unit:kilogram"
	UnitKilometerPerHour = "ht:unit:kph"
	UnitKilowattHour     = "ht:unit:kilowatthour"
	UnitLiter            = "ht:unit:liter"
	UnitLumen            = "ht:unit:lumen"
	UnitLux              = "ht:unit:lux"
	UnitMercury          = "ht:unit:mercury"
	UnitMeter            = "ht:unit:meter"
	UnitMeterPerSecond   = "ht:unit:meterspersecond"
	UnitMilesPerHour     = "ht:unit:milesperhour"
	UnitMilliSecond      = "ht:unit:millisecond"
	UnitMillibar         = "ht:unit:millibar"
	UnitMole             = "ht:unit:mole"
	UnitPSI              = "ht:unit:psi"
	UnitPascal           = "ht:unit:pascal"
	UnitPercent          = "ht:unit:percent"
	UnitPound            = "ht:unit:pound"
	UnitPpm              = "ht:unit:ppm"
	UnitRadian           = "ht:unit:radian"
	UnitSecond           = "ht:unit:second"
	UnitVolt             = "ht:unit:volt"
	UnitWatt             = "ht:unit:watt"
)

// end of UnitClasses

// UnitClassesMap maps @type to symbol, title and description
var UnitClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	UnitAmpere:           {Symbol: "A", Title: "Ampere", Description: "Electrical current in Amperes based on the elementary charge flow per second"},
	UnitKilogram:         {Symbol: "kg", Title: "Kilogram", Description: ""},
	UnitKilowattHour:     {Symbol: "kWh", Title: "Kilowatt-hour", Description: "non-SI unit of energy equivalent to 3.6 megajoules."},
	UnitLiter:            {Symbol: "l", Title: "Liter", Description: "SI unit of volume equivalent to 1 cubic decimeter."},
	UnitPascal:           {Symbol: "Pa", Title: "Pascal", Description: "SI unit of pressure. Equal to 1 newton of force applied over 1 square meter."},
	UnitLumen:            {Symbol: "lm", Title: "Lumen", Description: "SI unit luminous flux. Measure of perceived power of visible light. 1lm = 1 cd steradian"},
	UnitPpm:              {Symbol: "ppm", Title: "PPM", Description: "Parts per million"},
	UnitCandela:          {Symbol: "cd", Title: "Candela", Description: "SI unit of luminous intensity in a given direction. Roughly the same brightness as the common candle."},
	UnitDegree:           {Symbol: "degree", Title: "Degree", Description: "Angle in 0-360 degrees"},
	UnitMillibar:         {Symbol: "mbar", Title: "millibar", Description: "Metric unit of pressure. 1/1000th of a bar. Equal to 100 pascals. Amount of force it takes to move an object weighing a gram, one centimeter in one second."},
	UnitRadian:           {Symbol: "", Title: "Radian", Description: "Angle in 0-2pi"},
	UnitVolt:             {Symbol: "V", Title: "Volt", Description: "SI unit of electric potential; Energy consumption of 1 joule per electric charge of one coulomb"},
	UnitKilometerPerHour: {Symbol: "kph", Title: "Km per hour", Description: "Speed in kilometers per hour"},
	UnitMeterPerSecond:   {Symbol: "m/s", Title: "Meters per second", Description: "SI unit of speed in meters per second"},
	UnitMilliSecond:      {Symbol: "ms", Title: "millisecond", Description: "Unit of time in milli-seconds. Equal to 1/1000 of a second."},
	UnitPercent:          {Symbol: "%", Title: "Percent", Description: "Fractions of 100"},
	UnitKelvin:           {Symbol: "K", Title: "Kelvin", Description: "SI unit of thermodynamic temperature. 0 K represents absolute zero, the absence of all heat. 0 C equals +273.15K"},
	UnitMeter:            {Symbol: "m", Title: "Meter", Description: "Distance in meters. 1m=c/299792458"},
	UnitFahrenheit:       {Symbol: "F", Title: "Fahrenheit", Description: "Temperature in Fahrenheit"},
	UnitFoot:             {Symbol: "ft", Title: "Foot", Description: "Imperial unit of distance. 1 foot equals 0.3048 meters"},
	UnitPSI:              {Symbol: "PSI", Title: "PSI", Description: "Unit of pressure. Pounds of force per square inch. 1PSI equals 6984 Pascals."},
	UnitWatt:             {Symbol: "W", Title: "Watt", Description: "SI unit of power. Equal to 1 joule per second; or work performed when a current of 1 ampere flows across an electric potential of one volt."},
	UnitGallon:           {Symbol: "gl", Title: "Gallon", Description: "Unit of volume. 1 Imperial gallon is 4.54609 liters. 1 US liquid gallon is 3.78541 liters. 1 US dry gallon is 4.405 liters. "},
	UnitMercury:          {Symbol: "Hg", Title: "Mercury", Description: "Unit of atmospheric pressure in the United States. 1 Hg equals 33.8639 mbar."},
	UnitMilesPerHour:     {Symbol: "mph", Title: "Miles per hour", Description: "Speed in miles per hour"},
	UnitMole:             {Symbol: "mol", Title: "Mole", Description: "SI unit of measurement for amount of substance. Eg, molecules."},
	UnitPound:            {Symbol: "lbs", Title: "Pound", Description: "Imperial unit of weight. Equivalent to 0.453592 Kg. 1 Kg is 2.205 lbs"},
	UnitSecond:           {Symbol: "s", Title: "Second", Description: "SI unit of time based on caesium frequency"},
	UnitCelcius:          {Symbol: "C", Title: "Celcius", Description: "Temperature in Celcius"},
	UnitCount:            {Symbol: "(N)", Title: "Count", Description: ""},
	UnitLux:              {Symbol: "lx", Title: "Lux", Description: "SI unit illuminance. Equal to 1 lumen per square meter."},
}

// type: ActionClasses
// version: 0.1
// generated: 09 Jun 24 09:35 PDT
// source: github.com/hiveot/hub/api/vocab/ht-action-classes.yaml
// namespace: ht
const (
	ActionDimmer              = "ht:action:dimmer"
	ActionDimmerDecrement     = "ht:action:dimmer:decrement"
	ActionDimmerIncrement     = "ht:action:dimmer:increment"
	ActionDimmerSet           = "ht:action:dimmer:set"
	ActionMedia               = "ht:action:media"
	ActionMediaMute           = "ht:action:media:mute"
	ActionMediaNext           = "ht:action:media:next"
	ActionMediaPause          = "ht:action:media:pause"
	ActionMediaPlay           = "ht:action:media:play"
	ActionMediaPrevious       = "ht:action:media:previous"
	ActionMediaUnmute         = "ht:action:media:unmute"
	ActionMediaVolume         = "ht:action:media:volume"
	ActionMediaVolumeDecrease = "ht:action:media:volume:decrease"
	ActionMediaVolumeIncrease = "ht:action:media:volume:increase"
	ActionSwitch              = "ht:action:switch"
	ActionSwitchOff           = "ht:action:switch:off"
	ActionSwitchOn            = "ht:action:switch:on"
	ActionSwitchOnOff         = "ht:action:switch:onoff"
	ActionSwitchToggle        = "ht:action:switch:toggle"
	ActionThingDisable        = "ht:action:thing:disable"
	ActionThingEnable         = "ht:action:thing:enable"
	ActionThingStart          = "ht:action:thing:start"
	ActionThingStop           = "ht:action:thing:stop"
	ActionValveClose          = "ht:action:valve:close"
	ActionValveOpen           = "ht:action:valve:open"
)

// end of ActionClasses

// ActionClassesMap maps @type to symbol, title and description
var ActionClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	ActionDimmerSet:           {Symbol: "", Title: "Set dimmer", Description: "Action to set the dimmer value"},
	ActionMediaNext:           {Symbol: "", Title: "Next", Description: "Next track or station"},
	ActionMediaVolumeIncrease: {Symbol: "", Title: "Increase volume", Description: "Increase volume"},
	ActionSwitchOnOff:         {Symbol: "", Title: "Set On/Off switch", Description: "Action to set the switch on/off state"},
	ActionMediaVolumeDecrease: {Symbol: "", Title: "Decrease volume", Description: "Decrease volume"},
	ActionSwitch:              {Symbol: "", Title: "Switch", Description: "General switch action"},
	ActionThingStart:          {Symbol: "", Title: "Start", Description: "Start running a task"},
	ActionValveOpen:           {Symbol: "", Title: "Open valve", Description: "Action to open the valve"},
	ActionMediaPause:          {Symbol: "", Title: "Pause", Description: "Pause playback"},
	ActionDimmer:              {Symbol: "", Title: "Dimmer", Description: "General dimmer action"},
	ActionSwitchOff:           {Symbol: "", Title: "Switch off", Description: "Action to turn the switch off"},
	ActionValveClose:          {Symbol: "", Title: "Close valve", Description: "Action to close the valve"},
	ActionThingStop:           {Symbol: "", Title: "Stop", Description: "Stop a running task"},
	ActionMediaPrevious:       {Symbol: "", Title: "Previous", Description: "Previous track or station"},
	ActionMediaUnmute:         {Symbol: "", Title: "Unmute", Description: "Unmute audio"},
	ActionMediaVolume:         {Symbol: "", Title: "Volume", Description: "Set volume level"},
	ActionThingDisable:        {Symbol: "", Title: "Disable", Description: "Action to disable a thing"},
	ActionMedia:               {Symbol: "", Title: "Media control", Description: "Commands to control media recording and playback"},
	ActionMediaMute:           {Symbol: "", Title: "Mute", Description: "Mute audio"},
	ActionDimmerIncrement:     {Symbol: "", Title: "Increase dimmer", Description: ""},
	ActionThingEnable:         {Symbol: "", Title: "Enable", Description: "Action to enable a thing"},
	ActionMediaPlay:           {Symbol: "", Title: "Play", Description: "Start or continue playback"},
	ActionDimmerDecrement:     {Symbol: "", Title: "Lower dimmer", Description: ""},
	ActionSwitchOn:            {Symbol: "", Title: "Switch on", Description: "Action to turn the switch on"},
	ActionSwitchToggle:        {Symbol: "", Title: "Toggle switch", Description: "Action to toggle the switch"},
}

// type: PropertyClasses
// version: 0.1
// generated: 09 Jun 24 09:35 PDT
// source: github.com/hiveot/hub/api/vocab/ht-property-classes.yaml
// namespace: ht
const (
	PropAlarmMotion           = "ht:prop:alarm:motion"
	PropAlarmStatus           = "ht:prop:alarm:status"
	PropDevice                = "ht:prop:device"
	PropDeviceBattery         = "ht:prop:device:battery"
	PropDeviceDescription     = "ht:prop:device:description"
	PropDeviceEnabledDisabled = "ht:prop:device:enabled-disabled"
	PropDeviceFirmwareVersion = "ht:prop:device:firmwareversion"
	PropDeviceHardwareVersion = "ht:prop:device:hardwareversion"
	PropDeviceMake            = "ht:prop:device:make"
	PropDeviceModel           = "ht:prop:device:model"
	PropDevicePollinterval    = "ht:prop:device:pollinterval"
	PropDeviceSoftwareVersion = "ht:prop:device:softwareversion"
	PropDeviceStatus          = "ht:prop:device:status"
	PropDeviceTitle           = "ht:prop:device:title"
	PropElectric              = "ht:prop:electric"
	PropElectricCurrent       = "ht:prop:electric:current"
	PropElectricEnergy        = "ht:prop:electric:energy"
	PropElectricOverload      = "ht:prop:electric:overload"
	PropElectricPower         = "ht:prop:electric:poer"
	PropElectricVoltage       = "ht:prop:electric:voltage"
	PropEnv                   = "ht:prop:env"
	PropEnvAcceleration       = "ht:prop:env:acceleration"
	PropEnvAirquality         = "ht:prop:env:airquality"
	PropEnvBarometer          = "ht:prop:env:barometer"
	PropEnvCO                 = "ht:prop:env:co"
	PropEnvCO2                = "ht:prop:env:co2"
	PropEnvCpuload            = "ht:prop:env:cpuload"
	PropEnvDewpoint           = "ht:prop:env:dewpoint"
	PropEnvFuelFlowrate       = "ht:prop:env:fuel:flowrate"
	PropEnvFuelLevel          = "ht:prop:env:fuel:level"
	PropEnvHumidex            = "ht:prop:env:humidex"
	PropEnvHumidity           = "ht:prop:env:humidity"
	PropEnvLuminance          = "ht:prop:env:luminance"
	PropEnvPressure           = "ht:prop:env:pressure"
	PropEnvTemperature        = "ht:prop:env:temperature"
	PropEnvTimezone           = "ht:prop:env:timezone"
	PropEnvUV                 = "ht:prop:env:uv"
	PropEnvVibration          = "ht:prop:env:vibration"
	PropEnvVolume             = "ht:prop:env:volume"
	PropEnvWaterFlowrate      = "ht:prop:env:water:flowrate"
	PropEnvWaterLevel         = "ht:prop:env:water:level"
	PropEnvWindHeading        = "ht:prop:env:wind:heading"
	PropEnvWindSpeed          = "ht:prop:env:wind:speed"
	PropLocation              = "ht:prop:location"
	PropLocationCity          = "ht:prop:location:city"
	PropLocationLatitude      = "ht:prop:location:latitude"
	PropLocationLongitude     = "ht:prop:location:longitude"
	PropLocationName          = "ht:prop:location:name"
	PropLocationStreet        = "ht:prop:location:street"
	PropLocationZipcode       = "ht:prop:location:zipcode"
	PropMedia                 = "ht:prop:media"
	PropMediaMuted            = "ht:prop:media:muted"
	PropMediaPaused           = "ht:prop:media:paused"
	PropMediaPlaying          = "ht:prop:media:playing"
	PropMediaStation          = "ht:prop:media:station"
	PropMediaTrack            = "ht:prop:media:track"
	PropMediaVolume           = "ht:prop:media:volume"
	PropNet                   = "ht:prop:net"
	PropNetAddress            = "ht:prop:net:address"
	PropNetConnection         = "ht:prop:net:connection"
	PropNetDomainname         = "ht:prop:net:domainname"
	PropNetGateway            = "ht:prop:net:gateway"
	PropNetHostname           = "ht:prop:net:hostname"
	PropNetIP4                = "ht:prop:net:ip4"
	PropNetIP6                = "ht:prop:net:ip6"
	PropNetLatency            = "ht:prop:net:latency"
	PropNetMAC                = "ht:prop:net:mac"
	PropNetMask               = "ht:prop:net:mask"
	PropNetPort               = "ht:prop:net:port"
	PropNetSignalstrength     = "ht:prop:net:signalstrength"
	PropNetSubnet             = "ht:prop:net:subnet"
	PropStatusOnOff           = "ht:prop:status:onoff"
	PropStatusOpenClosed      = "ht:prop:status:openclosed"
	PropStatusStartedStopped  = "ht:prop:status:started-stopped"
	PropStatusYesNo           = "ht:prop:status:yes-no"
	PropSwitch                = "ht:prop:switch"
	PropSwitchDimmer          = "ht:prop:switch:dimmer"
	PropSwitchLight           = "ht:prop:switch:light"
	PropSwitchLocked          = "ht:prop:switch:locked"
	PropSwitchOnOff           = "ht:prop:switch:onoff"
)

// end of PropertyClasses

// PropertyClassesMap maps @type to symbol, title and description
var PropertyClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	PropNetConnection:         {Symbol: "", Title: "Connection", Description: "Connection status, connected, connecting, retrying, disconnected,..."},
	PropEnv:                   {Symbol: "", Title: "Environmental property", Description: "Property of environmental sensor"},
	PropLocationStreet:        {Symbol: "", Title: "Street", Description: "Street address"},
	PropEnvCpuload:            {Symbol: "", Title: "CPU load level", Description: "Device CPU load level"},
	PropEnvWaterFlowrate:      {Symbol: "", Title: "Water flow rate", Description: ""},
	PropLocation:              {Symbol: "", Title: "Location", Description: "General location information"},
	PropLocationLatitude:      {Symbol: "", Title: "Latitude", Description: "Latitude geographic coordinate"},
	PropNetMAC:                {Symbol: "", Title: "MAC", Description: "Hardware MAC address"},
	PropDevice:                {Symbol: "", Title: "Device attributes", Description: "Attributes describing a device"},
	PropElectricEnergy:        {Symbol: "", Title: "Energy", Description: "Electrical energy consumed"},
	PropEnvHumidex:            {Symbol: "", Title: "Humidex", Description: ""},
	PropLocationZipcode:       {Symbol: "", Title: "Zip code", Description: "Location ZIP code"},
	PropNetAddress:            {Symbol: "", Title: "Address", Description: "Network address"},
	PropNetIP6:                {Symbol: "", Title: "IP6 address", Description: "Device IP6 address"},
	PropElectricCurrent:       {Symbol: "", Title: "Current", Description: "Electrical current"},
	PropElectricPower:         {Symbol: "", Title: "Power", Description: "Electrical power being consumed"},
	PropLocationCity:          {Symbol: "", Title: "City", Description: "City name"},
	PropMediaTrack:            {Symbol: "", Title: "Track", Description: "Selected A/V track"},
	PropSwitchLight:           {Symbol: "", Title: "Light switch", Description: ""},
	PropElectricVoltage:       {Symbol: "", Title: "Voltage", Description: "Electrical voltage potential"},
	PropEnvCO:                 {Symbol: "", Title: "Carbon monoxide level", Description: "Carbon monoxide level"},
	PropDeviceTitle:           {Symbol: "", Title: "Title", Description: "Device friendly title"},
	PropEnvLuminance:          {Symbol: "", Title: "Luminance", Description: ""},
	PropDeviceFirmwareVersion: {Symbol: "", Title: "Firmware version", Description: ""},
	PropDeviceModel:           {Symbol: "", Title: "Model", Description: "Device model"},
	PropEnvVolume:             {Symbol: "", Title: "Volume", Description: ""},
	PropAlarmMotion:           {Symbol: "", Title: "Motion", Description: "Motion detected"},
	PropDeviceDescription:     {Symbol: "", Title: "Description", Description: "Device product description"},
	PropEnvFuelFlowrate:       {Symbol: "", Title: "Fuel flow rate", Description: ""},
	PropNet:                   {Symbol: "", Title: "Network properties", Description: "General network properties"},
	PropNetLatency:            {Symbol: "", Title: "Network latency", Description: "Delay between hub and client"},
	PropDeviceMake:            {Symbol: "", Title: "Make", Description: "Device manufacturer"},
	PropDeviceStatus:          {Symbol: "", Title: "Progress", Description: "Device status; alive, awake, dead, sleeping"},
	PropEnvWindSpeed:          {Symbol: "", Title: "Wind speed", Description: ""},
	PropEnvFuelLevel:          {Symbol: "", Title: "Fuel level", Description: ""},
	PropEnvTemperature:        {Symbol: "", Title: "Temperature", Description: ""},
	PropMediaMuted:            {Symbol: "", Title: "Muted", Description: "Audio is muted"},
	PropNetHostname:           {Symbol: "", Title: "Hostname", Description: "Hostname of the client"},
	PropSwitch:                {Symbol: "", Title: "Switch status", Description: ""},
	PropSwitchDimmer:          {Symbol: "", Title: "Dimmer value", Description: ""},
	PropDeviceEnabledDisabled: {Symbol: "", Title: "Enabled/Disabled", Description: "Enabled or disabled state"},
	PropEnvPressure:           {Symbol: "", Title: "Pressure", Description: ""},
	PropElectricOverload:      {Symbol: "", Title: "Overload protection", Description: "Cut load on overload"},
	PropEnvBarometer:          {Symbol: "", Title: "Atmospheric pressure", Description: "Barometric pressure of the atmosphere"},
	PropNetGateway:            {Symbol: "", Title: "Gateway", Description: "Network gateway address"},
	PropNetMask:               {Symbol: "", Title: "Netmask", Description: "Network mask. Example: 255.255.255.0 or 24/8"},
	PropStatusOnOff:           {Symbol: "", Title: "On/off status", Description: ""},
	PropStatusYesNo:           {Symbol: "", Title: "Yes/No", Description: "Progress with yes or no value"},
	PropDeviceBattery:         {Symbol: "", Title: "Battery level", Description: "Device battery level"},
	PropDeviceSoftwareVersion: {Symbol: "", Title: "Software version", Description: ""},
	PropMediaPaused:           {Symbol: "", Title: "Paused", Description: "Media is paused"},
	PropDeviceHardwareVersion: {Symbol: "", Title: "Hardware version", Description: ""},
	PropEnvAirquality:         {Symbol: "", Title: "Air quality", Description: "Air quality level"},
	PropLocationLongitude:     {Symbol: "", Title: "Longitude", Description: "Longitude geographic coordinate"},
	PropNetIP4:                {Symbol: "", Title: "IP4 address", Description: "Device IP4 address"},
	PropStatusStartedStopped:  {Symbol: "", Title: "Started/Stopped", Description: "Started or stopped status"},
	PropSwitchOnOff:           {Symbol: "", Title: "On/Off switch", Description: ""},
	PropAlarmStatus:           {Symbol: "", Title: "Alarm state", Description: "Current alarm status"},
	PropEnvUV:                 {Symbol: "", Title: "UV", Description: ""},
	PropEnvCO2:                {Symbol: "", Title: "Carbon dioxide level", Description: "Carbon dioxide level"},
	PropEnvVibration:          {Symbol: "", Title: "Vibration", Description: ""},
	PropMediaPlaying:          {Symbol: "", Title: "Playing", Description: "Media is playing"},
	PropMediaStation:          {Symbol: "", Title: "Station", Description: "Selected radio station"},
	PropNetDomainname:         {Symbol: "", Title: "Domain name", Description: "Domainname of the client"},
	PropSwitchLocked:          {Symbol: "", Title: "Lock", Description: "Electric lock status"},
	PropDevicePollinterval:    {Symbol: "", Title: "Polling interval", Description: "Interval to poll for updates"},
	PropEnvAcceleration:       {Symbol: "", Title: "Acceleration", Description: ""},
	PropNetSignalstrength:     {Symbol: "", Title: "Signal strength", Description: "Wireless signal strength"},
	PropStatusOpenClosed:      {Symbol: "", Title: "Open/Closed status", Description: ""},
	PropEnvDewpoint:           {Symbol: "", Title: "Dew point", Description: "Dew point temperature"},
	PropLocationName:          {Symbol: "", Title: "Location name", Description: "Name of the location"},
	PropEnvWaterLevel:         {Symbol: "", Title: "Water level", Description: ""},
	PropMediaVolume:           {Symbol: "", Title: "Volume", Description: "Media volume setting"},
	PropNetSubnet:             {Symbol: "", Title: "Subnet", Description: "Network subnet address. Example: 192.168.0.0"},
	PropEnvHumidity:           {Symbol: "", Title: "Humidity", Description: ""},
	PropEnvTimezone:           {Symbol: "", Title: "Timezone", Description: ""},
	PropMedia:                 {Symbol: "", Title: "Media commands", Description: "Control of media equipment"},
	PropNetPort:               {Symbol: "", Title: "Port", Description: "Network port"},
	PropElectric:              {Symbol: "", Title: "Electrical properties", Description: "General group of electrical properties"},
	PropEnvWindHeading:        {Symbol: "", Title: "Wind heading", Description: ""},
}
