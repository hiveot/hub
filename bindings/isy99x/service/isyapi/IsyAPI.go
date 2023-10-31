// Package internal with methods for talking to ISY99x nodes
package isyapi

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

// IsyAPI gateway access
type IsyAPI struct {
	address    string            // ISY IP address or file:// for simulation
	login      string            // Basic Auth login name
	password   string            // Basic Auth password
	simulation map[string]string // map used when in simulation
}

// IsyDevice Collection of ISY99x device information from multiple ISY REST calls
// Example:
// <configuration>
//
//	 <deviceSpecs>
//	     <make>Universal Nodes Inc.</make>
//	     <manufacturerURL>http://www.universal-devices.com</manufacturerURL>
//	     <model>Insteon Web Controller</model>
//	     <icon>/web/udlogo.jpg</icon>
//	     <archive>/web/insteon.jar</archive>
//	     <chart>/web/chart.jar</chart>
//	     <queryOnInit>true</queryOnInit>
//	     <oneNodeAtATime>true</oneNodeAtATime>
//	     <baseProtocolOptional>false</baseProtocolOptional>
//	 </deviceSpecs>
//	 <app>Insteon_UD99</app>
//	 <app_version>3.2.6</app_version>
//	 <platform>ISY-C-99</platform>
//	 <build_timestamp>2012-05-04-00:26:24</build_timestamp>
//	<root>
//	     <id>00:21:b9:01:0e:7b</id>
//	     <name>zzzz-donottouch</name>
//	 </root>
//	 <product>
//	     <id>1020</id>
//	     <desc>ISY 99i 256</desc>
//	 </product>
//	 ...
//
// </configuration>
type IsyDevice struct {
	Address string
	// GET /rest/config
	Configuration struct {
		DeviceSpecs struct {
			Make  string `xml:"make,omitempty"`
			Model string `xml:"model"`
			Icon  string `xml:"icon,,omitempty"`
		} `xml:"deviceSpecs"`
		App        string `xml:"app"`
		AppVersion string `xml:"app_version"`

		// controls describe the Value ID's (types)
		Controls struct {
			Control []struct {
				// control type; name field in controls
				ControlType string `xml:"name"`
				Label       string `xml:"label,omitempty"`
				// ReadOnly true fields show the status of the control: define as event
				// ReadOnly false fields can be set: define as action
				// Value is readonly(true)  or writable (false)
				ReadOnly    bool   `xml:"readOnly,omitempty"`    // default true
				IsQueryAble bool   `xml:"isQueryAble,omitempty"` // default false
				IsNumeric   bool   `xml:"isNumeric,omitempty"`
				NumericUnit string `xml:"numericUnit,omitempty"`
				Min         string `xml:"min,omitempty"`
				Max         string `xml:"max,omitempty"`
				IsGlobal    bool   `xml:"isGlobal,omitempty"`
				Actions     struct {
					Action []struct {
						Name        string `xml:"name"`  // name of the action
						Label       string `xml:"label"` // Human readable
						Description string `xml:"description,omitempty"`
						ReadOnly    bool   `xml:"readOnly,omitempty"`
					} `xml:"action,omitempty"`
				} `xml:"actions,omitempty"`
			} `xml:"control,omitempty"`
		} `xml:"controls,omitempty"`

		Platform       string `xml:"platform"`
		BuildTimestamp string `xml:"build_timestamp"`
		Root           struct {
			ID   string `xml:"id"`   // MAC
			Name string `xml:"name"` // customizable
		} `xml:"root"`
		Product struct {
			ID          string `xml:"id"`
			Description string `xml:"desc"`
		} `xml:"product"`
	} `xml:"configuration"`

	// GET /rest/sys
	System struct {
		MailTo          string `xml:"MailTo"`
		HTMLRole        int    `xml:"HTMLRole"`
		CompactEmail    bool   `xml:"CompactEmail"`
		QueryOnInit     bool   `xml:"QueryOnInit"`
		PCatchUp        bool   `xml:"PCatchUp"`
		PGracePeriod    int    `xml:"PGracePeriod"`
		WaitBusyReading bool   `xml:"WaitBusyReading"`
		NTPHost         string `xml:"NTPHost"`
		NTPActive       bool   `xml:"NTPActive"`
		NTPEnabled      bool   `xml:"NTPEnabled"`
		NTPInterval     int    `xml:"NTPInterval"`
	} `xml:"SystemOptions"`

	// GET /rest/time
	Time struct {
		NTP      int  `xml:"NTP"`
		TMOffset int  `xml:"TMOffset"`
		DST      bool `xml:"DST"`
		Sunrise  int  `xml:"Sunrise"`
		Sunset   int  `xml:"Sunset"`
	} `xml:"DT"`

	// GET /rest/network
	Network struct {
		Interface struct {
			IsDHCP  bool   `xml:"isDHCP,attr"` //
			IP      string `xml:"ip"`
			Mask    string `xml:"mask"`
			Gateway string `xml:"gateway"`
			DNS     string `xml:"dns"`
		} `xml:"Interface"`
		WebServer struct {
			HttpPort  string `xml:"HttpPort"`
			HttpsPort string `xml:"HttpsPort"`
		} `xml:"WebServer"`
	} `xml:"NetworkConfig"`
}

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

// IsyNode with info of a node on the gateway
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

	// Address is the unique node address. Used as thingID. Note that it contains spaces.
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
	// Property value
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

// ReadIsyGateway reads ISY gateway configuration
// returns isyDevice with device information
// See also: https://wiki.universal-devices.com/index.php?title=ISY_Developers:API:REST_Interface#Return_Values_/_Codes
func (isyAPI *IsyAPI) ReadIsyGateway() (isyDevice *IsyDevice, err error) {
	isyDevice = &IsyDevice{Address: isyAPI.address}
	err = isyAPI.isyRequest("/rest/config", &isyDevice.Configuration)
	if err == nil {
		err = isyAPI.isyRequest("/rest/sys", &isyDevice.System)
	}
	if err == nil {
		err = isyAPI.isyRequest("/rest/time", &isyDevice.Time)
	}
	if err == nil {
		err = isyAPI.isyRequest("/rest/network", &isyDevice.Network)
	}
	return isyDevice, err
}

// ReadIsyNodes reads the ISY Node list
func (isyAPI *IsyAPI) ReadIsyNodes() ([]*IsyNode, error) {
	isyNodes := IsyNodes{}

	err := isyAPI.isyRequest("/rest/nodes", &isyNodes)
	return isyNodes.Nodes, err
}

// ReadIsyStatus reads the ISY Node status
func (isyAPI *IsyAPI) ReadIsyStatus() (*IsyStatus, error) {
	isyStatus := IsyStatus{}
	err := isyAPI.isyRequest("/rest/status", &isyStatus)
	return &isyStatus, err
}

// WriteOnOff writes an on or off command to an isy node
// deviceID is the ISY node ID
// onOff is the new value to write
func (isyAPI *IsyAPI) WriteOnOff(deviceID string, onOff bool) error {
	var err error
	newValue := "DON"
	if onOff == false {
		newValue = "DOF"
	}
	// default to last set value
	isyAPI.simulation[deviceID] = newValue
	// can't request this in simulation mode
	if !strings.HasPrefix(isyAPI.address, "file://") {
		restPath := fmt.Sprintf("/rest/nodes/%s/cmd/%s", deviceID, newValue)
		err = isyAPI.isyRequest(restPath, nil)
	}
	return err
}

// isyRequest sends a request to the ISY device
// address contains the gateway address. If it starts with file:// then read from
// (simulation) file named <address>/<restPath>.xml
// restPath contains the REST url path for the request
func (isyAPI *IsyAPI) isyRequest(restPath string, response interface{}) error {
	// if address is a file then load content from file. Intended for testing
	if strings.HasPrefix(isyAPI.address, "file://") {
		filename := path.Join(isyAPI.address[7:], restPath+".xml")
		buffer, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("isyRequest: Unable to read ISY data from file", "filename", filename, "err", err)
			return err
		}
		err = xml.Unmarshal(buffer, &response)
		return err
	}

	// not a file, continue with http request
	isyURL := "http://" + isyAPI.address + restPath
	req, err := http.NewRequest("GET", isyURL, nil)

	if err != nil {
		return err
	}
	req.SetBasicAuth(isyAPI.login, isyAPI.password)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		slog.Error("isyRequest: Unable to read ISY device", "URL", isyURL, "err", err)
		return err
	} else if resp.StatusCode != 200 {
		msg := fmt.Sprintf("isyRequest: Error code return by ISY device %s: %v", isyURL, resp.Status)
		slog.Warn(msg)
		err = errors.New(msg)
		return err
	}

	// Decode the response into XML
	if response != nil {
		dec := xml.NewDecoder(resp.Body)
		_ = dec.Decode(&response)
		_ = resp.Body.Close()
	}

	return nil
}

// NewIsyAPI create an ISY gateway API instance.
// gwAddress is the ip address of the gateway, or "file://<path>" to a simulation xml file
// login to gateway device
// password to gateway device
func NewIsyAPI(gwAddress string, login string, password string) *IsyAPI {
	isy := &IsyAPI{}
	isy.address = gwAddress
	isy.login = login
	isy.password = password
	isy.simulation = make(map[string]string)
	return isy
}
