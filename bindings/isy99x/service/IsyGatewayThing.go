package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IsyGatewayThing represents the ISY gateway device of provides an API
// to invoke methods on the device.
// This implements IThing interface.
type IsyGatewayThing struct {
	// The Gateway is also a Thing, although it is not a node
	NodeThing

	// map of ISY product ID's
	prodMap map[string]InsteonProduct

	// The things that this gateway manages
	things map[string]*NodeThing

	// flag, a new node was discovered when reading values. Trigger a scan for new nodes.
	newNodeFound bool

	// protect access to the 'things' map
	mux sync.RWMutex

	//=== ISY Gateway Settings ===

	// GET /rest/config
	Configuration struct {
		DeviceSpecs struct {
			Make      string `xml:"make"`            // Universal Devices Inc.
			Manuf     string `xml:"manufacturerURL"` // http://www.universal-devices.com
			Model     string `xml:"model"`           // Insteon Web Controller
			Icon      string `xml:"icon"`            // /web/udlogo.jpg
			Archive   string `xml:"archive"`         // /web/insteon.jar
			Chart     string `xml:"chart"`           // /web/chart.jar
			QueryInit bool   `xml:"queryOnInit"`     // true
			OneNode   bool   `xml:"oneNodeAtATime"`  // true
		} `xml:"deviceSpecs"`
		App            string `xml:"app"`             // Insteon_UD99
		AppVersion     string `xml:"app_version"`     // 3.2.6
		Platform       string `xml:"platform"`        // ISY-C-99
		BuildTimestamp string `xml:"build_timestamp"` // 2012-05-04-00:26:24

		Root struct {
			ID   string `xml:"id"`   // MAC  aa:bb:cc:dd:ee:ff
			Name string `xml:"name"` // ISY gateway name customizable (might affect programs)
		} `xml:"root"`
		Product struct {
			ID          string `xml:"id"`   // 1020
			Description string `xml:"desc"` // ISY 99i 256
		} `xml:"product"`

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
		NTP       int  `xml:"NTP"`
		TMZOffset int  `xml:"TMZOffset"`
		DST       bool `xml:"DST"`
		Sunrise   int  `xml:"Sunrise"`
		Sunset    int  `xml:"Sunset"`
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

// AddIsyThing adds a representing of an Insteon device
func (igw *IsyGatewayThing) AddIsyThing(node *IsyNode) error {
	var isyThing *NodeThing
	var err error
	var thingID = node.Address

	parts := strings.Split(node.Type, ".")
	if len(parts) < 4 {
		return fmt.Errorf("AddIsyThing: expected 4 parts in node type: '%s'", node.Type)
	}
	cat, _ := strconv.ParseInt(parts[0], 10, 16)
	subCat, _ := strconv.ParseInt(parts[1], 10, 16)
	productID := fmt.Sprintf("0x%02X.0x%02X", cat, subCat)
	prodInfo := igw.prodMap[productID]
	hwVersion := parts[3]

	// determine what device this is using: <category.subcat.description.version>
	//deviceType, title := determineNodeDeviceType(node.Type)
	//the category determines the high level device type
	switch cat {
	case 2: // OnOff switch
		isyThing = &NewIsySwitchThing().NodeThing
		break
	case 7: // sensor switch
		isyThing = &NewIsySensorThing().NodeThing
		break
	default:
		isyThing = NewIsyThing()
	}
	if isyThing != nil {
		isyThing.Init(igw.ic, thingID, prodInfo, hwVersion)
		igw.things[thingID] = isyThing
	}
	return err
}

// GetIsyThing returns the ISY device Thing with the given ThingID
// Returns nil of a thing with this ID doesn't exist
func (igw *IsyGatewayThing) GetIsyThing(thingID string) things.IThing {
	igw.mux.RLock()
	defer igw.mux.RUnlock()
	it, _ := igw.things[thingID]
	return it
}

// GetIsyThings returns a list of ISY devices for publishing TD or values as updated in
// the last call to ReadIsyThings().
func (igw *IsyGatewayThing) GetIsyThings() []*NodeThing {
	igw.mux.RLock()
	defer igw.mux.RUnlock()
	things := make([]*NodeThing, 0, len(igw.things))
	for _, it := range igw.things {
		things = append(things, it)
	}
	return things
}

// GetTD returns the Gateway TD document
func (igw *IsyGatewayThing) GetTD() *things.TD {
	td := things.NewTD(igw.id, igw.Configuration.DeviceSpecs.Model, igw.deviceType)
	td.Description = igw.Configuration.DeviceSpecs.Make + "-" + igw.Configuration.DeviceSpecs.Model

	//--- device read-only attributes
	td.AddPropertyAsString(vocab.VocabManufacturer, vocab.VocabManufacturer, "Manufacturer")     // Universal Devices Inc.
	td.AddPropertyAsString(vocab.VocabModel, vocab.VocabModel, "Model")                          // ISY-C-99
	td.AddPropertyAsString(vocab.VocabSoftwareVersion, vocab.VocabSoftwareVersion, "AppVersion") // 3.2.6
	td.AddPropertyAsString(vocab.VocabMAC, vocab.VocabMAC, "MAC")                                // 00:21:xx:yy:... (mac)
	td.AddPropertyAsString(vocab.VocabProduct, vocab.VocabProduct, "Product")                    // ISY 99i 256
	td.AddPropertyAsString("ProductID", vocab.VocabProduct, "Product ID")                        // 1020
	prop := td.AddPropertyAsString("sunrise", "", "Sunrise")
	prop = td.AddPropertyAsString("sunset", "", "Sunset")

	//--- device configuration
	// custom name
	prop = td.AddPropertyAsString(vocab.VocabName, vocab.VocabName, "Name")
	prop.ReadOnly = false

	// network config
	prop = td.AddPropertyAsBool("DHCP", "", "DHCP enabled")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString(vocab.VocabLocalIP, vocab.VocabLocalIP, "IP address")
	prop.ReadOnly = igw.Network.Interface.IsDHCP == false
	prop = td.AddPropertyAsString("Gateway login name", "", "Login Name")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString("Gateway login password", "", "Password")
	prop.ReadOnly = false
	prop.WriteOnly = true

	// time config
	prop = td.AddPropertyAsString("NTPHost", "", "Network Time Host")
	prop.ReadOnly = false
	prop = td.AddPropertyAsBool("NTPEnabled", "", "NTP Enabled")
	prop.ReadOnly = false
	td.AddPropertyAsBool("DSTEnabled", "", "DST Enabled")
	prop.ReadOnly = false

	// TODO: any events?
	// TODO: any actions?

	return td
}

// Init initializes the gateway Thing for use and load the gateway configuration
func (igw *IsyGatewayThing) Init(ic *IsyConnection, id string, prodInfo InsteonProduct, hwVersion string) {
	igw.NodeThing.Init(ic, id, prodInfo, hwVersion)
	igw.deviceType = vocab.DeviceTypeGateway

	// values are used in TD title and description
	_ = igw.ReadGatewayValues()
}

// ReadGatewayValues reads ISY gateway properties.
// This loads the gateway 'Configuration', 'System', 'Time' and 'Network' data.
// See also: https://wiki.universal-devices.com/index.php?title=ISY_Developers:API:REST_Interface#Return_Values_/_Codes
func (igw *IsyGatewayThing) ReadGatewayValues() (err error) {

	const NTP_OFFSET = 2208988800

	err = igw.ic.SendRequest("GET", "/rest/config", &igw.Configuration)
	if err == nil {
		err = igw.ic.SendRequest("GET", "/rest/sys", &igw.System)
	}
	if err == nil {
		err = igw.ic.SendRequest("GET", "/rest/time", &igw.Time)
	}
	if err == nil {
		err = igw.ic.SendRequest("GET", "/rest/network", &igw.Network)
	}

	pv := igw.currentProps

	pv[vocab.VocabManufacturer] = igw.Configuration.DeviceSpecs.Make
	pv[vocab.VocabModel] = igw.Configuration.DeviceSpecs.Model
	pv[vocab.VocabSoftwareVersion] = igw.Configuration.AppVersion
	pv[vocab.VocabMAC] = igw.Configuration.Root.ID
	pv[vocab.VocabProduct] = igw.Configuration.Product.Description
	pv["ProductID"] = igw.Configuration.Product.ID
	// isy provides NTP stamp in local time, not in GMT :/
	sunrise := int64(igw.Time.Sunrise-igw.Time.TMZOffset) - NTP_OFFSET
	pv["sunrise"] = time.Unix(sunrise, 0).Format(time.TimeOnly) // seconds since epoc
	sunset := int64(igw.Time.Sunset-igw.Time.TMZOffset) - NTP_OFFSET
	pv["sunset"] = time.Unix(sunset, 0).Format(time.TimeOnly)     // seconds since epoc
	pv[vocab.VocabName] = igw.Configuration.Root.Name             // custom name
	pv["DHCP"] = strconv.FormatBool(igw.Network.Interface.IsDHCP) // true or false
	pv[vocab.VocabLocalIP] = igw.Network.Interface.IP
	pv[vocab.VocabPort] = igw.Network.WebServer.HttpPort
	//pv["Gateway login name"] = igw.LoginName
	pv["NTPHost"] = igw.System.NTPHost
	pv["NTPEnabled"] = strconv.FormatBool(igw.System.NTPEnabled)
	pv["DSTEnabled"] = strconv.FormatBool(igw.Time.DST)
	return err
}

// ReadIsyThings reads the ISY Node list and update the collection of ISY Things
func (igw *IsyGatewayThing) ReadIsyThings() error {
	isyNodes, err := igw.ic.ReadNodes()

	if err != nil {
		return err
	}
	for _, node := range isyNodes.Nodes {
		thingID := node.Address
		_, found := igw.things[thingID]
		if !found {
			err = igw.AddIsyThing(node)
			if err != nil {
				slog.Error("Error adding ISY device. Ignored.", "err", err)
			}
		}
	}
	return nil
}

// ReadIsyNodeValues reads the ISY Node status values and updates the nodes.
//
// This requests the status from the gateway and parses the response into a struct with
// xml properties as follows:
//
//	<node id="13 55 D3 1">
//	    <property id="ST" value="255" formatted="On" uom="on/off"/>
//	</node>
//
// Each ISY Thing will be updated with the latest status. It is up to them
// to notify their uses with an event if the status has changed.
func (igw *IsyGatewayThing) ReadIsyNodeValues() error {
	isyStatus := IsyStatus{}
	err := igw.ic.SendRequest("GET", "/rest/status", &isyStatus)
	for _, node := range isyStatus.Nodes {
		thingID := node.Address
		propID := node.Prop.ID
		newValue := node.Prop.Value
		uom := node.Prop.UOM

		it, found := igw.things[thingID]
		if found {
			err = it.HandleValueUpdate(propID, uom, newValue)
		} else {
			// new node found, refresh the node list
			igw.newNodeFound = true
		}
	}
	return err
}

// NewIsyGateway creates a new instance of the ISY gateway device representation.
// prodMap can be retrieved with LoadProductMapCSV()
// Call Init() before use.
func NewIsyGateway(prodMap map[string]InsteonProduct) *IsyGatewayThing {
	isyGW := &IsyGatewayThing{
		prodMap: prodMap,
		things:  make(map[string]*NodeThing),
	}
	return isyGW
}
