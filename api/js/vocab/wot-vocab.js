// Package vocab with WoT and JSON-LD defined vocabulary

// Thing Description document vocabulary definitions
// See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
// Thing dataschema Vocabulary


export const WoTAtType = "@type"

export const WoTAtContext = "@context"
export const WoTAnyURI = "https://www.w3.org/2019/wot/thing/v1"
export const WoTActions = "actions"
export const WoTCreated = "created"
export const WoTConst = "const"

export const WoTDataType = "type"
export const WoTDataTypeAnyURI = "anyURI"
export const WoTDataTypeArray = "array"
export const WoTDataTypeBool = "boolean"
export const WoTDataTypeDateTime = "dateTime"
export const WoTDataTypeInteger = "integer"
export const WoTDataTypeUnsignedInt = "unsignedInt"
export const WoTDataTypeNumber = "number"
export const WoTDataTypeObject = "object"
export const WoTDataTypeString = "string"
export const WoTDataTypeNone = ""

export const WoTDescription = "description"
export const WoTDescriptions = "descriptions"
export const WoTEnum = "enum"
export const WoTEvents = "events"
export const WoTFormat = "format"
export const WoTForms = "forms"
export const WoTHref = "href"
export const WoTID = "id"
export const WoTInput = "input"
export const WoTLinks = "links"
export const WoTMaximum = "maximum"
export const WoTMaxItems = "maxItems"
export const WoTMaxLength = "maxLength"
export const WoTMinimum = "minimum"
export const WoTMinItems = "minItems"
export const WoTMinLength = "minLength"
export const WoTModified = "modified"
export const WoTOperation = "op"
export const WoTOutput = "output"
export const WoTProperties = "properties"
export const WoTReadOnly = "readOnly"
export const WoTRequired = "required"
export const WoTSecurity = "security"
export const WoTSupport = "support"
export const WoTTitle = "title"
export const WoTTitles = "titles"
export const WoTVersion = "version"


// additional security schemas
// Intended for use by Hub services. HiveOT devices don't need them as they don't run a server
export const WoTNoSecurityScheme = "NoSecurityScheme"
export const WoTBasicSecurityScheme = "BasicSecurityScheme"
export const WoTDigestSecurityScheme = "DigestSecurityScheme"
export const WoTAPIKeySecurityScheme = "APIKeySecurityScheme"
export const WoTBearerSecurityScheme = "BearerSecurityScheme"
export const WoTPSKSecurityScheme = "PSKSecurityScheme"
export const WoTOAuth2SecurityScheme = "OAuth2SecurityScheme"

