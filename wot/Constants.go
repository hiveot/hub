package wot

// HiveOT operations that are missing in WoT and those needed by hub connected agents
// Keep in sync with vocab.yaml
const (
	HTOpPing  = "ping"
	HTOpError = "error"
)

// WoT operations
const (
	// actions
	OpCancelAction    = "cancelaction"
	OpInvokeAction    = "invokeaction"
	OpQueryAction     = "queryaction"
	OpQueryAllActions = "queryallactions"
	// events
	OpSubscribeAllEvents   = "subscribeallevents"
	OpSubscribeEvent       = "subscribeevent"
	OpUnsubscribeAllEvents = "unsubscribeallevents"
	OpUnsubscribeEvent     = "unsubscribeevent"
	// properties
	OpObserveAllProperties        = "observeallproperties"
	OpObserveMultipleProperties   = "observemultipleproperties"
	OpObserveProperty             = "observeproperty"
	OpReadAllProperties           = "readallproperties"
	OpReadMultipleProperties      = "readmultipleproperties"
	OpReadProperty                = "readproperty"
	OpUnobserveAllProperties      = "unobserveallproperties"
	OpUnobserveMultipleProperties = "unobservemultipleproperties"
	OpUnobserveProperty           = "unobserveproperty"
	OpWriteMultipleProperties     = "writemultipleproperties"
	OpWriteProperty               = "writeproperty"
)

// WoT data types
const (
	WoTDataType         = "type"
	DataTypeAnyURI      = "anyURI"
	DataTypeArray       = "array"
	DataTypeBool        = "boolean"
	DataTypeDateTime    = "dateTime"
	DataTypeInteger     = "integer"
	DataTypeNone        = ""
	DataTypeNumber      = "number"
	DataTypeObject      = "object"
	DataTypeString      = "string"
	DataTypeUnsignedInt = "unsignedInt"
)

// TD-1.1 affordance and data schema vocabulary
const (
	WoTActions              = "actions"
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
	WoTProperties           = "properties"
	WoTReadOnly             = "readOnly"
	WoTRequired             = "required"
	WoTSecurity             = "security"
	WoTSupport              = "support"
	WoTTitle                = "title"
	WoTTitles               = "titles"
	WoTVersion              = "version"
)
