package wot

// RFC3339Milli is the M2M format for ISO8601/RFC3339 timestamps in millisecond
const RFC3339Milli = "2006-01-02T15:04:05.999-0700"

// HiveOT operations that are missing in WoT and those needed by hub connected agents
const (
	HTOpLogin         = "login"
	HTOpLoginWithForm = "loginform"
	HTOpLogout        = "logout"
	HTOpPing          = "ping"
	HTOpPong          = "pong"
	HTOpError         = "error"
	// FIXME: opevent is replaced with subscribeevent operations
	HTOpEvent         = "event"
	HTOpReadAllEvents = "readallevents"
	HTOpReadAllTDs    = "readalltds"
	HTOpReadEvent     = "readevent"
	HTOpReadTD        = "readtd"
	HTOpRefresh       = "refresh"
	// FIXME: these are replaced with observeproperty operations
	HTOpUpdateMultipleProperties = "updatemultipleproperties"
	HTOpUpdateProperty           = "updateproperty"
	HTOpUpdateTD                 = "updatetd"
)

// WoT operations
const (
	OpCancelAction            = "cancelaction"
	OpInvokeAction            = "invokeaction"
	OpObserveAllProperties    = "observeallproperties"
	OpObserveProperty         = "observeproperty"
	OpQueryAction             = "queryaction"
	OpQueryAllActions         = "queryallactions"
	OpReadAllProperties       = "readallproperties"
	OpReadMultipleProperties  = "readmultipleproperties"
	OpReadProperty            = "readproperty"
	OpSubscribeAllEvents      = "subscribeallevents"
	OpSubscribeEvent          = "subscribeevent"
	OpUnobserveAllProperties  = "unobserveallproperties"
	OpUnobserveProperty       = "unobserveroperty"
	OpUnsubscribeAllEvents    = "unsubscribeallevents"
	OpUnsubscribeEvent        = "unsubscribeevent"
	OpWriteMultipleProperties = "writemultipleproperties"
	OpWriteProperty           = "writeproperty"
)

// WoT data types
const (
	WoTDataType            = "type"
	WoTDataTypeAnyURI      = "anyURI"
	WoTDataTypeArray       = "array"
	WoTDataTypeBool        = "boolean"
	WoTDataTypeDateTime    = "dateTime"
	WoTDataTypeInteger     = "integer"
	WoTDataTypeNone        = ""
	WoTDataTypeNumber      = "number"
	WoTDataTypeObject      = "object"
	WoTDataTypeString      = "string"
	WoTDataTypeUnsignedInt = "unsignedInt"
)

// TD-1.1 terminology
const (
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
	WoTReadOnly             = "readOnly"
	WoTRequired             = "required"
	WoTSecurity             = "security"
	WoTSupport              = "support"
	WoTTitle                = "title"
	WoTTitles               = "titles"
	WoTVersion              = "version"
)