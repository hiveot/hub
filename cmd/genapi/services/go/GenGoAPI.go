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
// * ThingID: "{Title(thingID)}AgentID"
// * Define types used in actions
// * Define a client function for each action
// * Define a service interface for handling an action
// * Define a message handler for invoking the service and returning a response
func GenGoAPIFromTD(td *things.TD, outfile string) (err error) {

	dThingID := td.ID
	agentID, serviceID := things.SplitDigiTwinThingID(dThingID)
	serviceName := ToTitle(serviceID)

	if agentID == "" {
		return fmt.Errorf("TD thingID does not have an agent prefix")
	}

	l := &utils.L{}
	l.Add("// Package %s with types and interfaces for using this service with agent '%s'",
		serviceID, agentID)
	l.Add("// DO NOT EDIT. This file is auto generated. Any changes will be overwritten.")
	l.Add("// Generated %s. ", time.Now().Format(time.RFC822))
	l.Add("package %s", serviceID)

	l.Add("")
	l.Add("import \"encoding/json\"")
	l.Add("import \"errors\"")
	l.Add("import \"github.com/hiveot/hub/runtime/api\"")
	l.Add("import \"github.com/hiveot/hub/lib/things\"")
	l.Add("import \"github.com/hiveot/hub/lib/hubclient\"")
	l.Add("")
	l.Add("// AgentID is the connection ID of the agent managing the Thing.")
	l.Add("const AgentID = \"%s\"", agentID)
	l.Add("// AgentID is the internal thingID of the device/service as used by agents.")
	l.Add("// Agents use this to publish events and subscribe to actions")
	l.Add("const AgentID = \"%s\"", serviceID)
	l.Add("// DThingID is the Digitwin thingID as used by agents. Digitwin adds the dtw:{agent} prefix to the serviceID")
	l.Add("// Consumers use this to publish actions and subscribe to events")
	l.Add("const DThingID = \"%s\"", dThingID)
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
