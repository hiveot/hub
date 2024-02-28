package service

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IsyGatewayThing is a Thing representing the ISY gateway device.
// This implements IThing interface.
type IsyGatewayThing struct {

	// REST/SOAP/WS connection to the ISY hub
	ic *IsyConnection

	// The gateway thing ID
	id string

	// map of ISY product ID's
	prodMap map[string]InsteonProduct

	// The things that this gateway manages
	things map[string]*NodeThing

	// flag, a new node was discovered when reading values. Trigger a scan for new nodes.
	newNodeFound bool

	// current property values of this thing
	propValues *things.PropertyValues

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
		isyThing.Init(igw.ic, node, prodInfo, hwVersion)
		igw.mux.Lock()
		igw.things[isyThing.GetID()] = isyThing
		igw.mux.Unlock()
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

// GetID return the gateway thingID
func (igw *IsyGatewayThing) GetID() string {
	return igw.id
}

// GetIsyThings returns a list of ISY devices for publishing TD or values as updated in
// the last call to ReadIsyThings().
func (igw *IsyGatewayThing) GetIsyThings() []*NodeThing {
	igw.mux.RLock()
	defer igw.mux.RUnlock()
	thingList := make([]*NodeThing, 0, len(igw.things))
	for _, it := range igw.things {
		thingList = append(thingList, it)
	}
	return thingList
}

// GetProps returns the current or changed properties.
// If changed properties are requested then clear the map of changes.
func (igw *IsyGatewayThing) GetProps(onlyChanges bool) map[string]string {
	return igw.propValues.GetValues(onlyChanges)
}

// GetTD returns the Gateway TD document
// This returns nil if the gateway wasn't initialized
func (igw *IsyGatewayThing) GetTD() *things.TD {
	if igw.ic == nil {
		return nil
	}

	td := things.NewTD(igw.id, igw.Configuration.DeviceSpecs.Model, vocab.ThingNetGateway)
	td.Description = igw.Configuration.DeviceSpecs.Make + "-" + igw.Configuration.DeviceSpecs.Model

	//--- device read-only attributes
	td.AddPropertyAsString(vocab.PropDeviceManufacturer, vocab.PropDeviceManufacturer, "Manufacturer")     // Universal Devices Inc.
	td.AddPropertyAsString(vocab.PropDeviceModel, vocab.PropDeviceModel, "Model")                          // ISY-C-99
	td.AddPropertyAsString(vocab.PropDeviceSoftwareVersion, vocab.PropDeviceSoftwareVersion, "AppVersion") // 3.2.6
	td.AddPropertyAsString(vocab.PropNetMAC, vocab.PropNetMAC, "MAC")                                      // 00:21:xx:yy:... (mac)
	td.AddPropertyAsString(vocab.PropDeviceDescription, vocab.PropDeviceDescription, "Product")            // ISY 99i 256
	td.AddPropertyAsString("ProductID", vocab.PropDeviceDescription, "Product ID")                         // 1020
	prop := td.AddPropertyAsString("sunrise", "", "Sunrise")
	prop = td.AddPropertyAsString("sunset", "", "Sunset")

	//--- device configuration
	// custom name
	prop = td.AddPropertyAsString(vocab.PropDeviceName, vocab.PropDeviceName, "Name")
	prop.ReadOnly = false

	// network config
	prop = td.AddPropertyAsBool("DHCP", "", "DHCP enabled")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString(vocab.PropNetIP4, vocab.PropNetIP4, "IP address")
	prop.ReadOnly = igw.Network.Interface.IsDHCP == false
	prop = td.AddPropertyAsString("Gateway login name", "", "Login Name")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString("Gateway login password", "", "Password")
	prop.ReadOnly = false
	prop.WriteOnly = true

	// time config
	prop = td.AddPropertyAsString("NTPHost", "", "Network time host")
	prop.ReadOnly = false
	prop.Default = "pool.ntp.org"
	prop = td.AddPropertyAsBool("NTPEnabled", "", "Use network time")
	prop.ReadOnly = false
	prop = td.AddPropertyAsInt("TMZOffset", "", "Timezone Offset")
	prop.ReadOnly = false
	prop.Unit = vocab.UnitSecond
	prop = td.AddPropertyAsBool("DSTEnabled", "", "DST Enabled")
	prop.ReadOnly = false

	// TODO: any events?

	// TODO: any actions?
	action := td.AddAction("StartLinking", "", "Start Linking",
		"1. Press and hold the 'Set' button on your Insteon device for 3-5 seconds.\n"+
			"2. Repeat step 1 for as many devices as you would like to link.\n"+
			"3. When done, select 'Finish' on the menu.", nil)
	_ = action
	action = td.AddAction("RemoveLink", "", "Remove Link",
		"1. Press and hold the 'Set' button on the Insteon device to unlink, for 3-5 seconds.\n"+
			"2. Repeat step 1 for as many devices as you would like to remove.\n"+
			"3. When done, select 'Finish' on the menu.", nil)
	_ = action
	return td
}

// Init re-initializes the gateway Thing for use and load the gateway configuration/
// This removes prior use nodes for a fresh start.
func (igw *IsyGatewayThing) Init(ic *IsyConnection) {
	igw.ic = ic
	igw.id = ic.GetID()
	igw.things = make(map[string]*NodeThing)
	igw.propValues = things.NewPropertyValues()

	// values are used in TD title and description
	_ = igw.ReadGatewayValues()
}

// ReadGatewayValues reads ISY gateway properties.
// This loads the gateway 'Configuration', 'System', 'Time' and 'Network' data.
// See also: https://wiki.universal-devices.com/index.php?title=ISY_Developers:API:REST_Interface#Return_Values_/_Codes
func (igw *IsyGatewayThing) ReadGatewayValues() (err error) {
	if igw.ic == nil {
		return fmt.Errorf("No ISY connection")
	}

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

	pv := igw.propValues

	pv.SetValue(vocab.PropDeviceManufacturer, igw.Configuration.DeviceSpecs.Make)
	pv.SetValue(vocab.PropDeviceModel, igw.Configuration.DeviceSpecs.Model)
	pv.SetValue(vocab.PropDeviceSoftwareVersion, igw.Configuration.AppVersion)
	pv.SetValue(vocab.PropNetMAC, igw.Configuration.Root.ID)
	pv.SetValue(vocab.PropDeviceDescription, igw.Configuration.Product.Description)
	pv.SetValue("ProductID", igw.Configuration.Product.ID)
	// isy provides NTP stamp in local time, not in GMT :/
	sunrise := int64(igw.Time.Sunrise-igw.Time.TMZOffset) - NTP_OFFSET
	pv.SetValue("sunrise", time.Unix(sunrise, 0).Format(time.TimeOnly))
	sunset := int64(igw.Time.Sunset-igw.Time.TMZOffset) - NTP_OFFSET
	pv.SetValue("sunset", time.Unix(sunset, 0).Format(time.TimeOnly))     // seconds since epoc
	pv.SetValue(vocab.PropDeviceName, igw.Configuration.Root.Name)        // custom name
	pv.SetValue("DHCP", strconv.FormatBool(igw.Network.Interface.IsDHCP)) // true or false
	pv.SetValue(vocab.PropNetIP4, igw.Network.Interface.IP)
	pv.SetValue(vocab.PropNetPort, igw.Network.WebServer.HttpPort)
	//pv.SetValue("Gateway login name",  igw.LoginName)
	pv.SetValue("NTPHost", igw.System.NTPHost)
	pv.SetValue("NTPEnabled", strconv.FormatBool(igw.System.NTPEnabled))
	pv.SetValue("TMZOffset", strconv.FormatInt(int64(igw.Time.TMZOffset), 10))
	pv.SetValue("DSTEnabled", strconv.FormatBool(igw.Time.DST))
	return err
}

// ReadIsyThings reads the ISY Node list and update the collection of ISY Things
func (igw *IsyGatewayThing) ReadIsyThings() error {
	if igw.ic == nil {
		return fmt.Errorf("No ISY connection")
	}
	isyNodes, err := igw.ic.ReadNodes()

	if err != nil {
		return err
	}
	for _, node := range isyNodes.Nodes {
		thingID := node.Address
		igw.mux.RLock()
		_, found := igw.things[thingID]
		igw.mux.RUnlock()
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
	if igw.ic == nil {
		return fmt.Errorf("No ISY connection")
	}

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
		prodMap:    prodMap,
		things:     make(map[string]*NodeThing),
		propValues: things.NewPropertyValues(),
	}
	return isyGW
}
