package eds

import (
	"strings"
	"time"
)

// ParseOneWireNodes parses the owserver xml data and returns a list of nodes,
// including the owserver gateway, and their parameters.
// This also converts sensor values to a proper decimals. Eg temperature isn't 4 digits but 1.
//
//	xmlNode is the node to parse, its attribute and possibly subnodes
//	latency to add to the root node (gateway device)
//	isRootNode is set for the first node, eg the gateway itself
func ParseOneWireNodes(
	xmlNode *XMLNode, latency time.Duration, isRootNode bool) []*OneWireNode {

	owNodeList := make([]*OneWireNode, 0)

	owNode := OneWireNode{
		// ID:          xmlNode.Attrs["ROMId"],
		Name:        xmlNode.XMLName.Local,
		Description: xmlNode.Description,
		Attr:        make(map[string]OneWireAttr),
		//DeviceType:  vocab.ThingNetGateway,
	}
	owNodeList = append(owNodeList, &owNode)
	// todo: find a better place for this
	//if isRootNode {
	//	owAttr := OneWireAttr{
	//		ID:       vocab.PropNetLatency,
	//		Name:     vocab.PropNetLatency,
	//		Value:    fmt.Sprintf("%.2f", latency.Seconds()),
	//		Unit:     "sec",
	//		DataType: vocab.WoTDataTypeNumber,
	//	}
	//	owNode.Attr[owAttr.Name] = owAttr
	//}
	// parse attributes and round sensor values
	for _, node := range xmlNode.Nodes {
		// if the xmlnode has no subnodes then it is a parameter describing the current node
		if len(node.Nodes) == 0 {
			writable := strings.ToLower(node.Writable) == "true"
			attrID := node.XMLName.Local

			unit := node.Units
			valueStr := string(node.Content)

			owAttr := OneWireAttr{
				ID:       attrID,
				Value:    valueStr,
				Unit:     unit,
				Writable: writable,
			}
			owNode.Attr[owAttr.ID] = owAttr
			// Family is used to determine device type
			if node.XMLName.Local == "Family" {
				owNode.Family = owAttr.Value
			} else if node.XMLName.Local == "ROMId" {
				// all subnodes use the ROMId as its ID
				owNode.ROMId = owAttr.Value
			} else if isRootNode && node.XMLName.Local == "DeviceName" {
				// The gateway itself uses the deviceName as its ID and name
				owNode.ROMId = owAttr.Value
				owNode.Name = owAttr.Value
				owNode.Description = "EDS OWServer Gateway"
				owNode.Attr["Manufacturer"] = OneWireAttr{
					ID:    "Manufacturer",
					Value: "EDS - Embedded Data Systems"}
				owNode.Attr["Model"] = OneWireAttr{
					ID:    "Model",
					Value: "OWServer V2"}
			}
		} else {
			// The node contains subnodes which contain one or more sensors.
			subNodes := ParseOneWireNodes(&node, 0, false)
			owNodeList = append(owNodeList, subNodes...)
		}
	}
	// owNode.ThingID = td.CreatePublisherThingID(pb.hubConfig.Zone, PluginID, owNode.NodeID, owNode.DeviceType)

	return owNodeList
}

// Apply the vocabulary to the name
// This returns the translated name from the vocabulary or the original name if not in the vocabulary
func applyVocabulary(name string, vocab map[string]string) (vocabName string, hasName bool) {
	vocabName, hasName = vocab[name]
	if !hasName {
		vocabName = name
	}
	return vocabName, hasName
}
