package eds

import (
	"encoding/xml"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// ReadEds reads EDS gateway and return the result as an XML node
// If edsAPI.address starts with file:// then read from file, otherwise from http
// If no address is configured, one will be auto discovered the first time.
func ReadEds(address, loginName, password string) (rootNode *XMLNode, err error) {
	// don't discover or read concurrently
	if strings.HasPrefix(address, "file://") {
		filename := address[7:]
		buffer, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("Unable to read EDS file", "err", err, "filename", filename)
			return nil, err
		}
		err = xml.Unmarshal(buffer, &rootNode)
		return rootNode, err
	}
	// not a file, continue with http request
	edsURL := address + "/details.xml"
	req, _ := http.NewRequest("GET", edsURL, nil)

	req.SetBasicAuth(loginName, password)
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)

	// resp, err := http.Get(edsURL)
	if err != nil {
		slog.Error("Unable to read EDS gateway", "err", err.Error(), "url", edsURL)
		return nil, err
	}
	// Decode the EDS response into XML
	dec := xml.NewDecoder(resp.Body)
	err = dec.Decode(&rootNode)
	_ = resp.Body.Close()

	return rootNode, err
}
