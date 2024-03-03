package eds

import "encoding/xml"

// XMLNode intermediate nested XML parsing node. Pure magic...
// --- https://stackoverflow.com/questions/30256729/how-to-traverse-through-xml-data-in-golang
type XMLNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []XMLNode  `xml:",any"`
	// Possible attributes for subnodes, depending on the property name
	Description string `xml:"Description,attr"`
	Writable    string `xml:"Writable,attr"`
	Units       string `xml:"Units,attr"`
}

// UnmarshalXML parse xml
func (n *XMLNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node XMLNode

	return d.DecodeElement((*node)(n), &start)
}

// OneWireNode with info on each node
type OneWireNode struct {
	//DeviceType string // derived from Family
	Family string
	// ThingID     string
	//NodeID      string // ROM ID
	ROMId       string
	Name        string // FIXME: Node model nr? title? can it be renamed?
	Description string
	Attr        map[string]OneWireAttr // attribute by ID
}

// OneWireAttr with info on each node attribute, property, event or action
type OneWireAttr struct {
	ID string // attribute instance ID, eg Name, Family, ROMId, ...
	//Name string // attribute Title for humans
	//VocabType  string // attribute type from vocabulary, if any, eg 'temperature', ...
	Unit     string
	Writable bool
	Value    string
	//IsActuator bool
	//IsSensor   bool   // sensors emit events on change
	//DataType string // vocab data type, "string", "number", "boolean", ""
}
