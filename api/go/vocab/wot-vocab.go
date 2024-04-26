// Package vocab with WoT and JSON-LD defined vocabulary
// This is not generated and can be edited by hand
package vocab

// Thing Description document vocabulary definitions
// See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition
// Thing, dataschema Vocabulary
const (
	WoTAtType              = "@type"
	WoTAtContext           = "@context"
	WoTAnyURI              = "https://www.w3.org/2019/wot/thing/v1"
	WoTActions             = "actions"
	WoTCreated             = "created"
	WoTConst               = "const"
	WoTDataType            = "type"
	WoTDataTypeAnyURI      = "anyURI"
	WoTDataTypeArray       = "array"
	WoTDataTypeBool        = "boolean"
	WoTDataTypeDateTime    = "dateTime"
	WoTDataTypeInteger     = "integer"
	WoTDataTypeUnsignedInt = "unsignedInt"
	WoTDataTypeNumber      = "number"
	WoTDataTypeObject      = "object"
	WoTDataTypeString      = "string"
	WoTDataTypeNone        = ""
	WoTDescription         = "description"
	WoTDescriptions        = "descriptions"
	WoTEnum                = "enum"
	WoTEvents              = "events"
	WoTFormat              = "format"
	WoTForms               = "forms"
	WoTHref                = "href"
	WoTID                  = "id"
	WoTInput               = "input"
	WoTLinks               = "links"
	WoTMaximum             = "maximum"
	WoTMaxItems            = "maxItems"
	WoTMaxLength           = "maxLength"
	WoTMinimum             = "minimum"
	WoTMinItems            = "minItems"
	WoTMinLength           = "minLength"
	WoTModified            = "modified"
	WoTOperation           = "op"
	WoTOutput              = "output"
	WoTProperties          = "properties"
	WoTReadOnly            = "readOnly"
	WoTRequired            = "required"
	WoTSecurity            = "security"
	WoTSupport             = "support"
	WoTTitle               = "title"
	WoTTitles              = "titles"
	WoTVersion             = "version"
)

// additional security schemas
// Intended for use by Hub services. HiveOT devices don't need them as they don't run a server
const (
	WoTNoSecurityScheme     = "NoSecurityScheme"
	WoTBasicSecurityScheme  = "BasicSecurityScheme"
	WoTDigestSecurityScheme = "DigestSecurityScheme"
	WoTAPIKeySecurityScheme = "APIKeySecurityScheme"
	WoTBearerSecurityScheme = "BearerSecurityScheme"
	WoTPSKSecurityScheme    = "PSKSecurityScheme"
	WoTOAuth2SecurityScheme = "OAuth2SecurityScheme"
)
