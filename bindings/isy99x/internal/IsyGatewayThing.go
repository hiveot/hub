package internal

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/internal/isy"
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
	*thing.ExposedThing

	// Configuration of this gateway thing
	config *config.Isy99xConfig

	// REST/SOAP/WS connection to the ISY hub
	gwAPI *isy.IsyGWAPI

	// product identification map by {cat}.{subcat}
	prodMap map[string]InsteonProduct

	// The isyThings that this gateway manages by thingID
	isyThings map[string]IIsyThing

	// protect access to the 'isyThings' map
	mux sync.RWMutex

	// flag, a new node was discovered when reading values. Trigger a scan for new nodes.
	newNodeFound bool

	//
	stopHeartbeatFn func()

	// current property values of this thing
	// propValues *ThingValues

	// callback handler for events
	evHandler IsyEventHandler

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
// If the node thing already exists it is updated.
func (svc *IsyGatewayThing) AddIsyThing(node *isy.IsyNode) (IIsyThing, error) {
	var isyThing IIsyThing
	var err error

	parts := strings.Split(node.Type, ".")
	if len(parts) < 4 {
		return nil, fmt.Errorf("AddIsyThing: expected 4 parts in node type: '%s'", node.Type)
	}
	cat, _ := strconv.ParseInt(parts[0], 10, 16)
	subCat, _ := strconv.ParseInt(parts[1], 10, 16)
	productID := fmt.Sprintf("0x%02X.0x%02X", cat, subCat)
	prodInfo := svc.prodMap[productID]
	hwVersion := parts[3]
	thingID := nodeID2ThingID(node.Address)

	// determine what device this is using: <category.subcat.description.version>
	//deviceType, title := determineNodeDeviceType(node.Type)
	//the category determines the high level device type
	switch cat {
	case 0: // general controller, tabletop/remote/touch panel
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 1: // dimmer control
		isyThing = NewIsyDimmerThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 2: // OnOff switch
		isyThing = NewIsySwitchThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 3: // network bridge
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 4: // irrigation control
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 5: // climate control
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 6: // pool/spa control
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 7: // sensor switch
		isyThing = NewIsySensorThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 9: // energy meter/management
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 14: // window/blinds
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	case 15: // access control/ door lock
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	default: // unknown general purpose thing
		isyThing = NewIsyThing(svc.gwAPI, thingID, node, prodInfo, hwVersion)
	}
	if isyThing != nil {
		// thingID := nodeID2ThingID(node.Address)
		// isyThing.Init(t.ic, thingID, node, prodInfo, hwVersion)
		svc.mux.Lock()
		svc.isyThings[isyThing.GetID()] = isyThing
		svc.mux.Unlock()
	}
	return isyThing, err
}

// GetIsyThing returns the ISY device Thing with the given ThingID
// Returns nil of a thing with this ID doesn't exist
func (svc *IsyGatewayThing) GetIsyThing(thingID string) IIsyThing {
	svc.mux.RLock()
	defer svc.mux.RUnlock()
	it, _ := svc.isyThings[thingID]
	return it
}

// GetIsyThingByNodeID returns the ISY device Thing with the given Node address
// Returns nil if a thing with this address doesn't exist
func (svc *IsyGatewayThing) GetIsyThingByNodeID(nodeAddr string) IIsyThing {
	thingID := nodeID2ThingID(nodeAddr)
	svc.mux.RLock()
	defer svc.mux.RUnlock()
	it, _ := svc.isyThings[thingID]
	return it
}

// GetIsyThings returns a list of ISY devices for publishing TD or values as updated in
// the last call to ReadIsyThings().
func (t *IsyGatewayThing) GetIsyThings() []IIsyThing {
	t.mux.RLock()
	defer t.mux.RUnlock()
	thingList := make([]IIsyThing, 0, len(t.isyThings))
	for _, it := range t.isyThings {
		thingList = append(thingList, it)
	}
	return thingList
}

// GetPropValues returns the current or changed property values.
// onlyChanges only provides changed properties
// func (t *IsyGatewayThing) GetPropValues(onlyChanges bool) map[string]any {
// 	state := t.Agent.GetState(t.GetThingID())
// 	values := state.GetAllProperties()
// 	// values := t.propValues.GetValues(onlyChanges)
// 	// TODO: add event values. Currently the TD does not list events.
// 	return values
// }

// HandleRequest passes the request to the associated Thing.
// This distributes the requests to the thing identified in the thingID
func (svc *IsyGatewayThing) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// request is either for this gateway or gateway nodes
	if req.ThingID == svc.GetID() {
		switch req.Operation {
		case wot.OpInvokeAction:
			// FIXME-1: how to determine the output of an action with ISY?
			// FIXME-2: how to determine an action has completed?
			// nothing to do
			err = fmt.Errorf("HandleGatewayRequest: Unknown action '%s'", req.Name)
		case wot.OpWriteProperty:
			err = svc.HandleConfigRequest(req, replyTo)
		default:
			// handle property read operations
			err = svc.ExposedThing.HandleRequest(req, replyTo)
		}
		return err
	} else {
		isyThing := svc.GetIsyThing(req.ThingID)
		if isyThing == nil {
			err := fmt.Errorf("HandleRequest: thing '%s' not found", req.ThingID)
			slog.Warn(err.Error())
			return err
		}
		return isyThing.HandleRequest(req, replyTo)
	}
}

// Handle property request for this device
func (svc *IsyGatewayThing) HandleConfigRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	slog.Info("HandleConfigRequest",
		slog.String("op", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("senderID", req.SenderID),
	)

	// FIXME: persist this configuration
	switch req.Name {
	case vocab.PropDevicePollinterval:
		err = fmt.Errorf("HandleConfigRequest: not implemented '%s'", req.Name)

	case vocab.PropNetAddress:
		svc.config.IsyAddress = req.ToString(0)
		_ = svc.gwAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)

	case "loginName":
		svc.config.LoginName = req.ToString(0)
		_ = svc.gwAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)

	case "password":
		svc.config.Password = req.ToString(0)
		_ = svc.gwAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)

	default:
		err = fmt.Errorf("HandleConfigRequest: Unknown config '%s'", req.Name)
	}
	return err
}

// MakeTD returns the Gateway TD document
func (svc *IsyGatewayThing) MakeTD() *wot.TD {

	tdoc := wot.NewTD(svc.GetID(), svc.Configuration.DeviceSpecs.Model, vocab.DeviceNetGateway)
	tdoc.Description = svc.Configuration.DeviceSpecs.Make + "-" + svc.Configuration.DeviceSpecs.Model

	//--- connection and polling info
	prop := tdoc.AddProperty(vocab.PropNetConnection, "Connected", "Device is connected", wot.DataTypeBool).
		SetAtType(vocab.PropNetConnection)
	prop.Description = "Whether the a connection to the ISY gateway exists"

	prop = tdoc.AddProperty(vocab.PropDevicePollinterval,
		"Poll Interval", "Poll for updates in seconds", wot.DataTypeInteger).
		SetAtType(vocab.PropDevicePollinterval)
	prop.Description = "Interval the binding polls the gateway for data value updates."
	prop.Unit = vocab.UnitSecond
	prop.ReadOnly = false

	// TBD: these are preconfigured or discovered
	// prop = tdoc.AddPropertyAsString("loginName", "Login name", "ISY99x gateway login name.")
	// prop.ReadOnly = false

	// prop = tdoc.AddPropertyAsString("password", "Password", "ISY99x gateway password")
	// prop.ReadOnly = false
	// prop.WriteOnly = true

	// prop = tdoc.AddProperty(vocab.PropNetAddress, "Network Address", "", wot.DataTypeString).
	// 	SetAtType(vocab.PropNetAddress)
	// prop.Description = "ISY99x gateway IP address; empty to auto discover."
	// prop.ReadOnly = false

	// events
	tdoc.AddEvent(vocab.PropNetConnection,
		"Connection changed", "Connection with ISY gateway device has changed",
		&wot.DataSchema{Type: wot.DataTypeBool},
	).SetAtType(vocab.PropNetConnection)

	//--- device read-only attributes
	tdoc.AddPropertyAsString(vocab.PropDeviceMake, "Manufacturer", "").
		SetAtType(vocab.PropDeviceMake) // Universal Devices Inc.
	tdoc.AddPropertyAsString(vocab.PropDeviceModel, "Model", "").
		SetAtType(vocab.PropDeviceModel) // ISY-C-99
	tdoc.AddPropertyAsString(vocab.PropDeviceSoftwareVersion, "AppVersion", "").
		SetAtType(vocab.PropDeviceSoftwareVersion) // 3.2.6
	tdoc.AddPropertyAsString(vocab.PropNetMAC, "MAC", "").
		SetAtType(vocab.PropNetMAC) // 00:21:xx:yy:... (mac)
	tdoc.AddPropertyAsString(vocab.PropDeviceDescription, "Product description", "").
		SetAtType(vocab.PropDeviceDescription) // ISY 99i 256
	tdoc.AddPropertyAsString("productID", "Product ID", "") // 1020
	prop = tdoc.AddPropertyAsString(PropIDSunrise, "Sunrise", "Current sunrise time").
		SetAtType(PropIDSunrise)
	prop = tdoc.AddPropertyAsString(PropIDSunset, "Sunset", "Current sunset time").
		SetAtType(PropIDSunset)

	//--- device configuration
	// custom name
	prop = tdoc.AddPropertyAsString(vocab.PropDeviceTitle, "Title", "").
		SetAtType(vocab.PropDeviceTitle)

	prop.ReadOnly = false

	// network config
	prop = tdoc.AddPropertyAsBool(PropIDDHCP, "DHCP enabled", "")
	prop.ReadOnly = false
	prop = tdoc.AddPropertyAsString(vocab.PropNetIP4, "IP4 address", "")
	prop.Description = "Configure gateway fix IP address"
	prop.ReadOnly = svc.Network.Interface.IsDHCP == false
	prop = tdoc.AddPropertyAsString(vocab.PropNetPort, "Port", "Gateway connection port")
	prop.ReadOnly = svc.Network.Interface.IsDHCP == false
	// prop = tdoc.AddPropertyAsString(PropIDLogin, "Login Name", "Gateway login name")
	// prop.ReadOnly = false
	// prop = tdoc.AddPropertyAsString(PropIDPassword, "Password", "Gateway password (hidden)")
	// prop.ReadOnly = false
	// prop.WriteOnly = true

	// time config
	prop = tdoc.AddPropertyAsString(PropIDNTPHost, "Network time host", "")
	prop.ReadOnly = false
	prop.Default = "pool.ntp.org"
	prop = tdoc.AddPropertyAsBool(PropIDNTPEnabled, "Use network time", "")
	prop.ReadOnly = false
	prop = tdoc.AddPropertyAsInt(PropIDTMZOffset, "Timezone Offset", "")
	prop.ReadOnly = false
	prop.Unit = vocab.UnitSecond
	prop = tdoc.AddPropertyAsBool(PropIDDSTEnabled, "DST Enabled", "")
	prop.ReadOnly = false

	// TODO: any events?

	// TODO: other actions?
	action := tdoc.AddAction("start-linking", "Start Linking",
		"1. Press and hold the 'Set' button on your Insteon device for 3-5 seconds.\n"+
			"2. Repeat step 1 for as many devices as you would like to link.\n"+
			"3. When done, select 'Finish' on the menu.", nil)
	_ = action
	action = tdoc.AddAction("remove-link", "Remove Link",
		"1. Press and hold the 'Set' button on the Insteon device to unlink, for 3-5 seconds.\n"+
			"2. Repeat step 1 for as many devices as you would like to remove.\n"+
			"3. When done, select 'Finish' on the menu.", nil)
	_ = action
	return tdoc
}

//
//// PubPropValues gets the thing properties and publish them
//func (svc *IsyGatewayThing) PubPropValues(hc hubclient.IAgent, onlyChanges bool) (err error) {
//	props := svc.GetPropValues(onlyChanges)
//	err = hc.PubMultipleProperties(svc.thingID, props)
//	return err
//}

// ReadIsyThings reads the ISY Node list and update the collection of ISY Things
func (svc *IsyGatewayThing) ReadIsyThings() error {
	if !svc.gwAPI.IsConnected() {
		return fmt.Errorf("No ISY connection")
	}
	// this reads all nodes
	isyNodes, err := svc.gwAPI.ReadNodes()

	if err != nil {
		return err
	}
	for _, node := range isyNodes.Nodes {
		it := svc.GetIsyThingByNodeID(node.Address)
		if it == nil {
			it, err = svc.AddIsyThing(node)
			if err != nil {
				slog.Error("Error adding ISY device. Ignored.", "err", err)
			}
		}
	}
	return nil
}

// RefreshProperties reads ISY gateway properties and publish changes.
// This loads the gateway 'Configuration', 'System', 'Time' and 'Network' data.
// See also: https://wiki.universal-devices.com/index.php?title=ISY_Developers:API:REST_Interface#Return_Values_/_Codes
func (svc *IsyGatewayThing) RefreshProperties(onlyChanges bool) (err error) {
	if !svc.gwAPI.IsConnected() {
		return fmt.Errorf("No gateway connection")
	}

	const NTP_OFFSET = 2208988800

	err = svc.gwAPI.SendRequest("GET", "/rest/config", "", &svc.Configuration)
	if err == nil {
		err = svc.gwAPI.SendRequest("GET", "/rest/sys", "", &svc.System)
	}
	if err == nil {
		err = svc.gwAPI.SendRequest("GET", "/rest/time", "", &svc.Time)
	}
	if err == nil {
		err = svc.gwAPI.SendRequest("GET", "/rest/network", "", &svc.Network)
	}

	thingID := svc.GetID()
	sunrise := int64(svc.Time.Sunrise-svc.Time.TMZOffset) - NTP_OFFSET
	sunset := int64(svc.Time.Sunset-svc.Time.TMZOffset) - NTP_OFFSET

	props := map[string]any{
		vocab.PropDeviceMake:            svc.Configuration.DeviceSpecs.Make,
		vocab.PropDeviceModel:           svc.Configuration.DeviceSpecs.Model,
		vocab.PropDeviceSoftwareVersion: svc.Configuration.AppVersion,
		vocab.PropNetMAC:                svc.Configuration.Root.ID,
		vocab.PropDeviceDescription:     svc.Configuration.Product.Description,
		wot.WoTTitle:                    svc.Configuration.Root.Name, // custom name
		vocab.PropNetIP4:                svc.Network.Interface.IP,
		vocab.PropNetPort:               svc.Network.WebServer.HttpPort,

		"productID": svc.Configuration.Product.ID,
		PropIDDHCP:  strconv.FormatBool(svc.Network.Interface.IsDHCP), // true or false
		//pv.SetValue(PropIDLogin, igw.Configuration.LoginName)

		// isy provides NTP stamp in local time, not in GMT!
		PropIDSunrise: time.Unix(sunrise, 0).Format(time.TimeOnly),
		PropIDSunset:  time.Unix(sunset, 0).Format(time.TimeOnly), // seconds since epoc

		PropIDNTPHost:    svc.System.NTPHost,
		PropIDNTPEnabled: strconv.FormatBool(svc.System.NTPEnabled),
		PropIDTMZOffset:  strconv.FormatInt(int64(svc.Time.TMZOffset), 10),
		PropIDDSTEnabled: strconv.FormatBool(svc.Time.DST),
	}
	svc.PubProperties(thingID, props, onlyChanges)

	return err
}

// RefreshStatus reads the ISY Node status values and updates the nodes.
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
func (svc *IsyGatewayThing) RefreshStatus() error {
	if !svc.gwAPI.IsConnected() {
		return fmt.Errorf("No ISY connection")
	}

	isyStatus := isy.IsyStatus{}
	err := svc.gwAPI.SendRequest("GET", "/rest/status", "", &isyStatus)
	for _, node := range isyStatus.Nodes {
		propID := node.Prop.ID
		newValue := node.Prop.Value
		uom := node.Prop.UOM

		it := svc.GetIsyThingByNodeID(node.Address)
		if it != nil {
			err = it.HandleValueUpdate(propID, uom, newValue)
		} else {
			// new node found, refresh the node list
			svc.newNodeFound = true
		}
	}
	return err
}

// Start publishes the gateway TD, its properties and start the heartbeat
func (svc *IsyGatewayThing) Start() {
	tdoc := svc.MakeTD()
	svc.WriteTD(tdoc)

	svc.stopHeartbeatFn = svc.startHeartbeat()
}

// heartbeat polls the gateway device every X seconds and publishes updates
// This returns a stop function that can be used to end the loop
func (svc *IsyGatewayThing) startHeartbeat() (stopFn func()) {

	var tdCountDown = 0
	var pollCountDown = 0
	var republishCountDown = 0
	var isConnected = false
	var forceRepublish bool
	var err error
	var wasConnected bool

	stopFn = utils.StartHeartbeat(time.Second, func() {
		tdCountDown--
		pollCountDown--
		republishCountDown--
		forceRepublish = false

		// if no gateway connection exists, try to reestablish a connection to the gateway
		isConnected = svc.gwAPI.IsConnected()
		if !isConnected {
			// if the connection dropped, send an event
			if wasConnected {
				svc.PubEvent("", vocab.PropNetConnection, isConnected)
			}
			err = svc.gwAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
			// if err == nil {
			// 	// re-establish the gateway device connection
			// 	svc.IsyGW.Init(svc.isyAPI)
			// }
			isConnected = svc.gwAPI.IsConnected()
			if isConnected {
				svc.PubEvent("", vocab.PropNetConnection, isConnected)
			}
		}
		wasConnected = isConnected

		// publish node TDs and values
		if isConnected && tdCountDown <= 0 {
			tdoc := svc.MakeTD()
			svc.WriteTD(tdoc)
			tdCountDown = svc.config.TDInterval
			// after publishing the TD, republish all values
			forceRepublish = true
		}
		// publish changes to sensor/actuator values
		if isConnected && pollCountDown <= 0 {
			// publish changed values or periodically for publishing all values
			if republishCountDown <= 0 {
				republishCountDown = svc.config.RepublishInterval
				forceRepublish = true
			}
			err = svc.RefreshProperties(!forceRepublish)
			for _, thing := range svc.isyThings {
				thing.RefreshProperties(!forceRepublish)
			}
			pollCountDown = svc.config.PollInterval
			// slow down if this fails. Don't flood the logs
			if err != nil {
				pollCountDown = svc.config.PollInterval * 5
			}
		}
	})
	return stopFn
}

// Stop the heartbeat
func (svc *IsyGatewayThing) Stop() {
	slog.Info("Stopping the ISY99x gateway service")
	//svc.isRunning.Store(false)
	svc.mux.Lock()
	defer svc.mux.Unlock()
	if svc.stopHeartbeatFn != nil {
		svc.stopHeartbeatFn()
		svc.stopHeartbeatFn = nil
	}
	// let processes finish
	time.Sleep(time.Millisecond)
}

// NewIsyGateway creates a new instance of the ISY gateway device.
// This connects to the gateway and returns an error if the gateway is unreachable.
func NewIsyGateway(appID string, cfg *config.Isy99xConfig) (*IsyGatewayThing, error) {

	prodMap, err := LoadProductMapCSV("")
	_ = err

	// connect to the ISY99x gateway, if possible.
	// this can fail so let it run.
	gwAPI := isy.NewIsyAPI()
	err = gwAPI.Connect(cfg.IsyAddress, cfg.LoginName, cfg.Password)

	// if err != nil {
	// 	return nil, err
	// }
	thingID := gwAPI.GetID()

	isyGW := &IsyGatewayThing{
		ExposedThing: thing.NewExposedThing(thingID, nil),
		config:       cfg,
		prodMap:      prodMap,
		isyThings:    make(map[string]IIsyThing),
		gwAPI:        gwAPI,
	}

	return isyGW, nil
}
