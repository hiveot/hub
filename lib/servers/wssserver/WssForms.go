package wssserver

import (
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

// AddTDForms adds forms for use of this protocol to the given TD.
//
// Since the contentType is the default application/json it is omitted
//
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// This is a massive waste of space in the TD.
func (srv *WssServer) AddTDForms(tdoc *td.TD, includeAffordances bool) {

	// 1 form for all operations
	form := td.NewForm("", srv.GetConnectURL(), SubprotocolWSS)
	form["op"] = []string{
		wot.OpQueryAllActions,
		wot.OpObserveAllProperties, wot.OpUnobserveAllProperties,
		wot.OpReadAllProperties, // wot.OpWriteMultipleProperties,
		wot.OpSubscribeAllEvents, wot.OpUnsubscribeAllEvents,
	}
	//form["contentType"] = "application/json"

	tdoc.Forms = append(tdoc.Forms, form)

	// Add forms to all affordances to be compliant with the specifications.
	// This is a massive waste of space in the TD.
	if includeAffordances {
		srv.AddAffordanceForms(tdoc)
	}
}

// AddAffordanceForms adds forms to affordances for interacting using the websocket protocol binding
func (srv *WssServer) AddAffordanceForms(tdoc *td.TD) {
	href := srv.GetConnectURL()
	for name, aff := range tdoc.Actions {
		_ = name
		form := td.NewForm("", href, SubprotocolWSS)
		form["op"] = []string{wot.OpInvokeAction, wot.OpQueryAction}
		aff.AddForm(form)
		// cancel action is currently not supported
	}
	for name, aff := range tdoc.Events {
		_ = name
		form := td.NewForm("", href, SubprotocolWSS)
		form["op"] = []string{wot.OpSubscribeEvent, wot.OpUnsubscribeEvent}
		aff.AddForm(form)
	}
	for name, aff := range tdoc.Properties {
		_ = name
		form := td.NewForm("", href, SubprotocolWSS)
		ops := []string{}
		if !aff.WriteOnly {
			ops = append(ops, wot.OpReadProperty, wot.OpObserveProperty, wot.OpUnobserveProperty)
		}
		if !aff.ReadOnly {
			ops = append(ops, wot.OpWriteProperty)
		}

		form["op"] = ops
		aff.AddForm(form)

	}
}
