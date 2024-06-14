package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/isy99x/service/isy"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Non-vocab property IDs
const (
	PropIDDHCP       = "DHCP"
	PropIDDSTEnabled = "DSTEnabled"
	PropIDLogin      = "login"
	PropIDNTPHost    = "NTPHost"
	PropIDNTPEnabled = "NTPEnabled"
	PropIDPassword   = "password"
	PropIDSunrise    = "sunrise"
	PropIDSunset     = "sunset"
	PropIDTMZOffset  = "TMZOffset"
)

// IsyGatewayThing is a Thing representing the ISY gateway device.
// This implements IThing interface.
type IsyGatewayThing struct {

	// REST/SOAP/WS connection to the ISY hub
	ic *isy.IsyAPI

	// The gateway thingID
	thingID string

	// map of ISY product ID's
	prodMap map[string]InsteonProduct

	// The things that this gateway manages by thingID
	things map[string]IIsyThing

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
			ID   string `xml:"nodeID"` // MAC  aa:bb:cc:dd:ee:ff
			Name string `xml:"name"`   // ISY gateway name customizable (might affect programs)
		} `xml:"root"`
		Product struct {
			ID          string `xml:"nodeID"` // 1020
			Description string `xml:"desc"`   // ISY 99i 256
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

// derive a thingID from a ISY nodeID
// ISY node IDs have spaces in them, which are not allowed in Thing IDs
func nodeID2ThingID(nodeID string) string {
	thingID := strings.ReplaceAll(nodeID, " ", "-")
	return thingID
}

// AddIsyThing adds a representing of an Insteon device
func (igw *IsyGatewayThing) AddIsyThing(node *isy.IsyNode) error {
	var isyThing IIsyThing
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
	case 0: // general controller, tabletop/remote/touch panel
		isyThing = NewIsyThing()
		break
	case 1: // dimmer control
		isyThing = NewIsyDimmerThing()
		break
	case 2: // OnOff switch
		isyThing = NewIsySwitchThing()
		break
	case 3: // network bridge
		isyThing = NewIsyThing()
		break
	case 4: // irrigation control
		isyThing = NewIsyThing()
		break
	case 5: // climate control
		isyThing = NewIsyThing()
		break
	case 6: // pool/spa control
		isyThing = NewIsyThing()
		break
	case 7: // sensor switch
		isyThing = NewIsySensorThing()
		break
	case 9: // energy meter/management
		isyThing = NewIsyThing()
		break
	case 14: // window/blinds
		isyThing = NewIsyThing()
		break
	case 15: // access control/ door lock
		isyThing = NewIsyThing()
		break
	default: // unknown general purpose thing
		isyThing = NewIsyThing()
	}
	if isyThing != nil {
		thingID := nodeID2ThingID(node.Address)
		isyThing.Init(igw.ic, thingID, node, prodInfo, hwVersion)
		igw.mux.Lock()
		igw.things[isyThing.GetID()] = isyThing
		igw.mux.Unlock()
	}
	return err
}

// GetIsyThing returns the ISY device Thing with the given ThingID
// Returns nil of a thing with this ID doesn't exist
func (igw *IsyGatewayThing) GetIsyThing(thingID string) IIsyThing {
	igw.mux.RLock()
	defer igw.mux.RUnlock()
	it, _ := igw.things[thingID]
	return it
}

// GetIsyThingByNodeID returns the ISY device Thing with the given Node address/ID
// Returns nil if a thing with this ID doesn't exist
func (igw *IsyGatewayThing) GetIsyThingByNodeID(nodeID string) IIsyThing {
	thingID := nodeID2ThingID(nodeID)
	igw.mux.RLock()
	defer igw.mux.RUnlock()
	it, _ := igw.things[thingID]
	return it
}

// GetID return the gateway thingID
func (igw *IsyGatewayThing) GetID() string {
	return igw.thingID
}

// GetIsyThings returns a list of ISY devices for publishing TD or values as updated in
// the last call to ReadIsyThings().
func (igw *IsyGatewayThing) GetIsyThings() []IIsyThing {
	igw.mux.RLock()
	defer igw.mux.RUnlock()
	thingList := make([]IIsyThing, 0, len(igw.things))
	for _, it := range igw.things {
		thingList = append(thingList, it)
	}
	return thingList
}

// GetValues returns the current or changed property and event values.
// onlyChanges only provides changed properties and event values
func (igw *IsyGatewayThing) GetValues(onlyChanges bool) (map[string]string, map[string]string) {
	values := igw.propValues.GetValues(onlyChanges)
	// TODO: add event values. Currently the TD does not list events.
	events := make(map[string]string)
	return values, events
}

// GetTD returns the Gateway TD document
// This returns nil if the gateway wasn't initialized
func (igw *IsyGatewayThing) GetTD() *things.TD {
	if igw.ic == nil {
		return nil
	}

	td := things.NewTD(igw.thingID, igw.Configuration.DeviceSpecs.Model, vocab.ThingNetGateway)
	td.Description = igw.Configuration.DeviceSpecs.Make + "-" + igw.Configuration.DeviceSpecs.Model

	//--- device read-only attributes
	td.AddPropertyAsString("", vocab.PropDeviceMake, "Manufacturer")               // Universal Devices Inc.
	td.AddPropertyAsString("", vocab.PropDeviceModel, "Model")                     // ISY-C-99
	td.AddPropertyAsString("", vocab.PropDeviceSoftwareVersion, "AppVersion")      // 3.2.6
	td.AddPropertyAsString("", vocab.PropNetMAC, "MAC")                            // 00:21:xx:yy:... (mac)
	td.AddPropertyAsString("", vocab.PropDeviceDescription, "Product description") // ISY 99i 256
	td.AddPropertyAsString("ProductID", "", "Product ID")                          // 1020
	prop := td.AddPropertyAsString(PropIDSunrise, "", "Sunrise")
	prop = td.AddPropertyAsString(PropIDSunset, "", "Sunset")

	//--- device configuration
	// custom name
	prop = td.AddPropertyAsString("", vocab.PropDeviceTitle, "Title")
	prop.ReadOnly = false

	// network config
	prop = td.AddPropertyAsBool(PropIDDHCP, "", "DHCP enabled")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString("", vocab.PropNetIP4, "IP address")
	prop.Description = "Configure gateway fix IP address"
	prop.ReadOnly = igw.Network.Interface.IsDHCP == false
	prop = td.AddPropertyAsString(PropIDLogin, "", "Gateway login name")
	prop.ReadOnly = false
	prop = td.AddPropertyAsString(PropIDPassword, "", "Gateway password (hidden)")
	prop.ReadOnly = false
	prop.WriteOnly = true

	// time config
	prop = td.AddPropertyAsString(PropIDNTPHost, "", "Network time host")
	prop.ReadOnly = false
	prop.Default = "pool.ntp.org"
	prop = td.AddPropertyAsBool(PropIDNTPEnabled, "", "Use network time")
	prop.ReadOnly = false
	prop = td.AddPropertyAsInt(PropIDTMZOffset, "", "Timezone Offset")
	prop.ReadOnly = false
	prop.Unit = vocab.UnitSecond
	prop = td.AddPropertyAsBool(PropIDDSTEnabled, "", "DST Enabled")
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
func (igw *IsyGatewayThing) Init(ic *isy.IsyAPI) {
	igw.ic = ic
	igw.thingID = ic.GetID()
	igw.things = make(map[string]IIsyThing)
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

	err = igw.ic.SendRequest("GET", "/rest/config", "", &igw.Configuration)
	if err == nil {
		err = igw.ic.SendRequest("GET", "/rest/sys", "", &igw.System)
	}
	if err == nil {
		err = igw.ic.SendRequest("GET", "/rest/time", "", &igw.Time)
	}
	if err == nil {
		err = igw.ic.SendRequest("GET", "/rest/network", "", &igw.Network)
	}

	pv := igw.propValues

	pv.SetValue(vocab.PropDeviceMake, igw.Configuration.DeviceSpecs.Make)
	pv.SetValue(vocab.PropDeviceModel, igw.Configuration.DeviceSpecs.Model)
	pv.SetValue(vocab.PropDeviceSoftwareVersion, igw.Configuration.AppVersion)
	pv.SetValue(vocab.PropNetMAC, igw.Configuration.Root.ID)
	pv.SetValue(vocab.PropDeviceDescription, igw.Configuration.Product.Description)
	//pv.SetValue(vocab.PropDeviceDescription, igw.Configuration.Product.Description)
	pv.SetValue(vocab.PropDeviceTitle, igw.Configuration.Root.Name) // custom name
	pv.SetValue(vocab.PropNetIP4, igw.Network.Interface.IP)
	pv.SetValue(vocab.PropNetPort, igw.Network.WebServer.HttpPort)

	pv.SetValue("ProductID", igw.Configuration.Product.ID)
	pv.SetValue(PropIDDHCP, strconv.FormatBool(igw.Network.Interface.IsDHCP)) // true or false
	//pv.SetValue(PropIDLogin, igw.Configuration.LoginName)

	// isy provides NTP stamp in local time, not in GMT!
	sunrise := int64(igw.Time.Sunrise-igw.Time.TMZOffset) - NTP_OFFSET
	pv.SetValue(PropIDSunrise, time.Unix(sunrise, 0).Format(time.TimeOnly))
	sunset := int64(igw.Time.Sunset-igw.Time.TMZOffset) - NTP_OFFSET
	pv.SetValue(PropIDSunset, time.Unix(sunset, 0).Format(time.TimeOnly)) // seconds since epoc

	pv.SetValue(PropIDNTPHost, igw.System.NTPHost)
	pv.SetValue(PropIDNTPEnabled, strconv.FormatBool(igw.System.NTPEnabled))
	pv.SetValue(PropIDTMZOffset, strconv.FormatInt(int64(igw.Time.TMZOffset), 10))
	pv.SetValue(PropIDDSTEnabled, strconv.FormatBool(igw.Time.DST))
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
		it := igw.GetIsyThingByNodeID(node.Address)
		if it == nil {
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
//	<node nodeID="13 55 D3 1">
//	    <property nodeID="ST" value="255" formatted="On" uom="on/off"/>
//	</node>
//
// Each ISY Thing will be updated with the latest status. It is up to them
// to notify their uses with an event if the status has changed.
func (igw *IsyGatewayThing) ReadIsyNodeValues() error {
	if igw.ic == nil {
		return fmt.Errorf("No ISY connection")
	}

	isyStatus := isy.IsyStatus{}
	err := igw.ic.SendRequest("GET", "/rest/status", "", &isyStatus)
	for _, node := range isyStatus.Nodes {
		propID := node.Prop.ID
		newValue := node.Prop.Value
		uom := node.Prop.UOM

		it := igw.GetIsyThingByNodeID(node.Address)
		if it != nil {
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
		things:     make(map[string]IIsyThing),
		propValues: things.NewPropertyValues(),
	}
	return isyGW
}
