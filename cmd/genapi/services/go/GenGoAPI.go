package _go

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"time"
)

// TODO: this service ID should be defined in the service
const DigitwinServiceID = "digitwin"

// GenGoAPIFromTD generates a golang source file from a Thing Description Document of a service.
// Intended for services.
//
// TODO: use TD forms, if defined, to link to a protocol.
// Currently this uses a provided message transport that implements the transportation protocol
// to talk to the Hub. All actions are invoked through this transport.
//
// This generates:
// * ThingID: "{Title(thingID)}ServiceID"
// * Define types used in actions
// * Define a client function for each action
// * Define a service interface for handling an action
// * Define a message handler for invoking the service and returning a response
func GenGoAPIFromTD(td *things.TD, outfile string) (err error) {

	l := &utils.L{}
	l.Add("// Package %s with types and interfaces for using this service", td.ID)
	l.Add("// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.")
	l.Add("// Generated %s. ", time.Now().Format(time.RFC822))
	l.Add("package %s", td.ID)

	l.Add("")
	l.Add("import \"encoding/json\"")
	l.Add("import \"fmt\"")
	l.Add("import \"github.com/hiveot/hub/runtime/api\"")
	l.Add("import \"github.com/hiveot/hub/lib/things\"")
	l.Add("")
	l.Add("// the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix")
	l.Add("const RawThingID = \"%s\"", td.ID)
	l.Add("const ThingID = \"urn:%s:%s\"", DigitwinServiceID, td.ID)
	l.Add("")
	GenActionStructs(l, td)
	GenActionClient(l, td)
	GenServiceInterface(l, td)
	GenServiceHandler(l, td)

	if l.Size() > 0 {
		err = l.Write(outfile)
	}
	return err
}
