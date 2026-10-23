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
	Family      string
	ROMId       string
	Name        string // FIXME: Node model nr? title? can it be renamed?
	Description string
	Attr        map[string]OneWireAttr // attribute by ID
}

// OneWireAttr with info on each node key, unit, and value
type OneWireAttr struct {
	// ID is the attribute instance ID, eg Name, Family, ROMId, ...
	ID       string
	Unit     string
	Writable bool
	Value    string
}
