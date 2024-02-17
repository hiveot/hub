package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/huin/goupnp"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// GET /rest/config
type ISYConfiguration struct {
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
	//} `xml:"configuration"`
}

// IsyConnection manages the connection to the ISY99x gateway
// This supports publishing REST and Soap requests.
// Use of WebSockets is tbd
type IsyConnection struct {
	// address contains the IP address of the ISY gateway, or a path to a file.
	// if address starts with file:// then this is used to read from a
	// (simulation) file named <address>/<restPath>.xml
	address  string
	login    string // Basic Auth login name
	password string // Basic Auth password
	//simulation map[string]string // map used when in simulation

	id string
}

// Connect to the gateway and read its make and model
func (ic *IsyConnection) Connect(
	gwAddr string, login string, password string) (err error) {

	if gwAddr == "" {
		gwAddr, err = ic.Discover(time.Second * 3)
		if err != nil {
			return err
		}
	}
	ic.address = gwAddr
	ic.login = login
	ic.password = password

	// get the ISY gateway ID and test if it is reachable
	cfg := ISYConfiguration{}
	err = ic.SendRequest("GET", "/rest/config", &cfg)
	ic.id = cfg.App
	//ic.id = cfg.Product.ID

	// TODO: websocket connect
	return err
}

func (ic *IsyConnection) Disconnect() {
	// TODO: websocket disconnect
}

// Discover any ISY99x gateway on the local network for 3 seconds.
// If successful this returns the ISY99x gateway address, otherwise an error is returned.
func (ic *IsyConnection) Discover(timeout time.Duration) (addr string, err error) {
	slog.Info("Starting discovery")
	deviceType := "urn:udi-com:device:X_Insteon_Lighting_Device:1"
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	devices, err := goupnp.DiscoverDevicesCtx(ctx, deviceType /*ssdp.SSDPAll*/)
	cancelFn()

	if len(devices) == 0 {
		slog.Info("No ISY99x device found")
		return "", fmt.Errorf("no ISY99x device found")
	}
	// success!
	d0 := devices[0]
	slog.Info("Success! Found ISY99x device.",
		"addr", d0.Location.Host,
		"name", d0.Root.Device.FriendlyName,
		"manufacturer", d0.Root.Device.Manufacturer,
		"modelName", d0.Root.Device.ModelName,
		"modelNumber", d0.Root.Device.ModelNumber,
	)

	return d0.Location.Host, err
}

// GetID returns the ISY Gateway ID
func (ic *IsyConnection) GetID() string {
	return ic.id
}

// ReadNodes reads the ISY Node list
func (ic *IsyConnection) ReadNodes() (*IsyNodes, error) {
	isyNodes := IsyNodes{}
	err := ic.SendRequest("GET", "/rest/nodes", &isyNodes)
	return &isyNodes, err
}

// SendRequest sends a GET or PUT request to the ISY device
//
//	method is the GET or POST method to use.
//	restPath contains REST or SOAP path to send the request to.
//	If it starts with file:// then read from (simulation) file named <address>/<restPath>.xml
func (ic *IsyConnection) SendRequest(method string, restPath string, response interface{}) error {
	// if address is a file then load content from file. Intended for testing
	if strings.HasPrefix(ic.address, "file://") {
		filename := path.Join(ic.address[7:], restPath+".xml")
		buffer, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("isyRequest: Unable to read ISY data from file", "filename", filename, "err", err)
			return err
		}
		err = xml.Unmarshal(buffer, &response)
		return err
	}

	// not a file, continue with http request
	isyURL := "http://" + ic.address + restPath
	req, err := http.NewRequest(method, isyURL, nil)

	if err != nil {
		return err
	}
	req.SetBasicAuth(ic.login, ic.password)
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

// NewIsyConnection create an ISY connection handler instance.
//
//	gwAddress is the ip address of the gateway, or "file://<path>" to a simulation xml file
//	login to gateway device
//	password to gateway device
func NewIsyConnection() *IsyConnection {
	isy := &IsyConnection{}
	//isy.simulation = make(map[string]string)
	return isy
}
