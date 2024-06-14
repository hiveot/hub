package isy

// IsyNodes Collection of ISY99x nodes. Example:
// <nodes>
//
//	<root>Nodes</root>
//	<node flag="128">
//	    <address>13 55 D3 1</address>
//	    <name>Basement</name>
//	    <parent type="3">49025</parent>
//	    <type>2.12.56.0</type>
//	    <enabled>true</enabled>
//	    <pnode>13 55 D3 1</pnode>
//	    <ELK_ID>A04</ELK_ID>
//	    <property id="ST" value="255" formatted="On" uom="on/off"/>
//	</node>
type IsyNodes struct {
	// The name of ISY network. If blank, then it may be construed as the Network node.
	Root string `xml:"root,omitempty"`
	// ignore the folder names
	Nodes []*IsyNode `xml:"node"`
}

// IsyNode is an immutable object temporarily holding data about an ISY device.
// It is created from the data returned by the ISY gateway and used to create or
// update the IsyThing instance.
type IsyNode struct {
	// Flag value map
	//NODE_IS_INIT 0x01 //needs to be initialized
	//NODE_TO_SCAN 0x02 //needs to be scanned
	//
	//NODE_IS_A_GROUP 0x04 //it’s a group!
	//NODE_IS_ROOT 0x08 //it’s the root group
	//NODE_IS_IN_ERR 0x10 //it’s in error!
	//NODE_IS_NEW 0x20 //brand new node
	//NODE_TO_DELETE 0x40 //has to be deleted later
	//NODE_IS_DEVICE_ROOT 0x80 //root device such as KPL load
	Flag uint `xml:"flag,attr"` //

	// Address is the unique node address.
	// Note that it contains spaces which are not allowed in a Thing ID.
	Address string `xml:"address"`
	// Name is the friendly name of the node (writable)
	Name string `xml:"name"`
	// Parent contains the address of the parent (not used)
	Parent struct {
		Value string `xml:",chardata"`
		// Type is the parent type:
		// NODE_TYPE_NOTSET 0 (unknown)
		// NODE_TYPE_NODE 1
		// NODE_TYPE_GROUP 2
		// NODE_TYPE_FOLDER 3
		Type string `xml:"type,attr"`
	} `xml:"parent"`
	//Type is the type of device.
	//For INSTEON: {device-cat}.{device-subcat}.{version}.{reserved} (4 bytes)
	// Please note that the if family element is present, then the device type is a
	// category/sub category within the family.
	// In most cases the device cat/subcat for all families are the same
	Type string `xml:"type"`
	// Node can be enabled/disabled (writable)
	Enabled string `xml:"enabled"`
	//The address of the primary node for the device partially represented by this node.
	//If this node is the primary node then pnode will equal address.
	//A device may be represented by one or more nodes.
	//One of these nodes is designated the primary node and is used to help group the set of nodes for a device.
	//Note: UPB Mandatory/INSTEON Optional.
	Pnode string `xml:"pnode,omitempty"`
	// Property holding the node's value
	Property IsyProp `xml:"property"`
}

// IsyStatus with status as returned by the controller. Example:
// <nodes>
//
//	<node id="13 55 D3 1">
//	    <property id="ST" value="255" formatted="On" uom="on/off"/>
//	</node>
//	<node id="13 57 73 1">
//	    <property id="ST" value="255" formatted="On" uom="on/off"/>
//	</node>
//	...
type IsyStatus struct {
	Nodes []struct {
		Address string  `xml:"id,attr"`  // The ID attribute is the actual ISY node address
		Prop    IsyProp `xml:"property"` // TODO: Can a node haves multiple properties with their values?
	} `xml:"node"`
}

// IsyProp with a status property value
type IsyProp struct {
	ID        string `xml:"id,attr"`    // Property ID: ST, OL, ...
	Value     string `xml:"value,attr"` //
	Formatted string `xml:"formatted,attr"`
	UOM       string `xml:"uom,attr"`
}
