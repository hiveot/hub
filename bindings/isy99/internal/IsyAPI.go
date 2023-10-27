// Package internal with methods for talking to ISY99x nodes
package internal

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
//    <deviceSpecs>
//        <make>Universal Nodes Inc.</make>
//        <manufacturerURL>http://www.universal-devices.com</manufacturerURL>
//        <model>Insteon Web Controller</model>
//        <icon>/web/udlogo.jpg</icon>
//        <archive>/web/insteon.jar</archive>
//        <chart>/web/chart.jar</chart>
//        <queryOnInit>true</queryOnInit>
//        <oneNodeAtATime>true</oneNodeAtATime>
//        <baseProtocolOptional>false</baseProtocolOptional>
//    </deviceSpecs>
//    <app>Insteon_UD99</app>
//    <app_version>3.2.6</app_version>
//    <platform>ISY-C-99</platform>
//    <build_timestamp>2012-05-04-00:26:24</build_timestamp>
//   <root>
//        <id>00:21:b9:01:0e:7b</id>
//        <name>zzzz-donottouch</name>
//    </root>
//    <product>
//        <id>1020</id>
//        <desc>ISY 99i 256</desc>
//    </product>
//    ...
// </configuration>
type IsyDevice struct {
	Configuration struct {
		DeviceSpecs struct {
			Make  string `xml:"make"`
			Model string `xml:"model"`
		} `xml:"deviceSpecs"`
		App            string `xml:"app"`
		AppVersion     string `xml:"app_version"`
		Platform       string `xml:"platform"`
		BuildTimestamp string `xml:"build_timestamp"`
		Root           struct {
			ID string `xml:"id"` // MAC
		} `xml:"root"`
		Product struct {
			ID          string `xml:"id"`
			Description string `xml:"desc"`
		} `xml:"product"`
	} `xml:"configuration"`
	// network struct {
	// 	Interface struct {
	// 		DHCP    string `xml:"isDHCP,attr"`
	// 		IP      string `xml:"ip"`
	// 		Mask    string `xml:"mask"`
	// 		Gateway string `xml:"gateway"`
	// 		DNS     string `xml:"dns"`
	// 	} `xml:"Interface"`
	// }
}

// IsyNodes Collection of ISY99x nodes. Example:
// <nodes>
//    <root>Nodes</root>
//    <node flag="128">
//        <address>13 55 D3 1</address>
//        <name>Basement</name>
//        <parent type="3">49025</parent>
//        <type>2.12.56.0</type>
//        <enabled>true</enabled>
//        <pnode>13 55 D3 1</pnode>
//        <ELK_ID>A04</ELK_ID>
//        <property id="ST" value="255" formatted="On" uom="on/off"/>
//    </node>
type IsyNodes struct {
	// ignore the folder names
	Nodes []*IsyNode `xml:"node"`
}

// IsyNode with info of a node on the gateway
type IsyNode struct {
	Address  string  `xml:"address"`
	Name     string  `xml:"name"`
	Parent   string  `xml:"parent"`
	Type     string  `xml:"type"`
	Enabled  string  `xml:"enabled"`
	Pnode    string  `xml:"pnode"`
	Property IsyProp `xml:"property"`
}

// IsyStatus with status as returned by the controller. Example:
// <nodes>
//    <node id="13 55 D3 1">
//        <property id="ST" value="255" formatted="On" uom="on/off"/>
//    </node>
//    <node id="13 57 73 1">
//        <property id="ST" value="255" formatted="On" uom="on/off"/>
//    </node>
//    ...
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

// ReadIsyStatus reads the ISY Node status
func (isyAPI *IsyAPI) ReadIsyStatus() (*IsyStatus, error) {
	isyStatus := IsyStatus{}
	err := isyAPI.isyRequest("/rest/status", &isyStatus)
	return &isyStatus, err
}

// ReadIsyNodes reads the ISY Node list
func (isyAPI *IsyAPI) ReadIsyNodes() (*IsyNodes, error) {
	isyNodes := IsyNodes{}

	err := isyAPI.isyRequest("/rest/nodes", &isyNodes)
	return &isyNodes, err
}

// ReadIsyGateway reads ISY gateway configuration and status
// returns isyDevice with device information
func (isyAPI *IsyAPI) ReadIsyGateway() (isyDevice *IsyDevice, err error) {
	isyDevice = &IsyDevice{}
	err = isyAPI.isyRequest("/rest/config", &isyDevice.Configuration)
	return isyDevice, err
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
		buffer, err := ioutil.ReadFile(filename)
		if err != nil {
			logrus.Errorf("isyRequest: Unable to read ISY data from file from %s: %v", filename, err)
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
		logrus.Warnf("pollDevice: Unable to read ISY device from %s: %v", isyURL, err)
		return err
	} else if resp.StatusCode != 200 {
		msg := fmt.Sprintf("pollDevice: Error code return by ISY device %s: %v", isyURL, resp.Status)
		logrus.Warn(msg)
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

// NewIsyAPI create an ISY API proxy
// gatewayAddress is the ip address of the gateway, or "file://<path>" to a simulation xml file
// login to gateway device
// password to gateway device
func NewIsyAPI(gatewayAddress string, login string, password string) *IsyAPI {
	isy := &IsyAPI{}
	isy.address = gatewayAddress
	isy.login = login
	isy.password = password
	isy.simulation = make(map[string]string)
	return isy
}
