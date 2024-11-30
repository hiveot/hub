package wssserver

import (
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

func (svc *WssTransportServer) AddTDForms(td *tdd.TD) error {
	svc.AddThingLevelForms(td)
	//svc.AddPropertiesForms(td)
	//svc.AddEventsForms(td)
	//svc.AddActionForms(td)
	return nil
}

// GetForm returns a new form for a websocket supported operation
// Intended for Thing level operations
func (svc *WssTransportServer) GetForm(op string) tdd.Form {
	// map operations to message type

	msgType, found := svc.op2MsgType[op]
	if !found {
		slog.Error("GetForm. Operation doesn't have corresponding message type",
			"op", op)
		return nil
	}
	form := tdd.Form{}
	form["op"] = op
	form["subprotocol"] = "websocket"
	form["contentType"] = "application/json"
	form["href"] = svc.wssPath
	form["messageType"] = msgType

	return form
}

// AddThingLevelForms adds forms with protocol info to the TD, and its properties, events and actions
// HiveOT mostly uses top level forms.
func (svc *WssTransportServer) AddThingLevelForms(td *tdd.TD) {
	// iterate the thing level operations
	//params := map[string]string{"thingID": td.ID}

	// supported message type and operations
	//for msgType, op := range wssbinding.MsgTypeToOp {
	//	_ = msgType
	//	form := svc.GetForm(op)
	//	td.Forms = append(td.Forms, form)
	//}

	// apparently you can just add 1 form containing all operations...
	// still struggling with this stuff.
	form := tdd.Form{}
	form["op"] = svc.opList
	form["subprotocol"] = "websocket"
	form["contentType"] = "application/json"
	form["href"] = svc.wssPath
	td.Forms = append(td.Forms, form)

}
