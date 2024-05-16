package _go

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"time"
)

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

	agentID, nativeThingID, valid := things.SplitDigiTwinThingID(td.ID)
	serviceName := ToTitle(nativeThingID)

	if !valid {
		return fmt.Errorf("TD thingID does not have an agent prefix")
	}

	l := &utils.L{}
	l.Add("// Package %s with types and interfaces for using this service with agent '%s'",
		nativeThingID, agentID)
	l.Add("// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.")
	l.Add("// Generated %s. ", time.Now().Format(time.RFC822))
	l.Add("package %s", nativeThingID)

	l.Add("")
	l.Add("import \"encoding/json\"")
	l.Add("import \"errors\"")
	l.Add("import \"github.com/hiveot/hub/runtime/api\"")
	l.Add("import \"github.com/hiveot/hub/lib/things\"")
	l.Add("import \"github.com/hiveot/hub/lib/hubclient\"")
	l.Add("")
	l.Add("// RawThingID is the raw thingID as used by agents. Digitwin adds the urn:{agent} prefix")
	l.Add("const RawThingID = \"%s\"", nativeThingID)
	l.Add("const ThingID = \"%s\"", td.ID)
	l.Add("")
	GenActionStructs(l, td)
	GenActionClient(l, td)
	GenServiceInterface(l, serviceName, td)
	GenServiceHandler(l, serviceName, td)

	if l.Size() > 0 {
		err = l.Write(outfile)
	}
	return err
}
