package wssserver

import (
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
)

// AddTDForms adds forms for use of this protocol to the given TD
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// This is a massive waste of space in the TD.
func (srv *WssServer) AddTDForms(tdoc *td.TD, includeAffordances bool) {

	// 1 form for all operations
	form := td.NewForm(wot.OpQueryAllActions, srv.GetConnectURL(), SubprotocolWSS)
	form["op"] = []string{
		wot.OpQueryAllActions,
		wot.OpObserveAllProperties, wot.OpUnobserveAllProperties,
		wot.OpReadAllProperties, // wot.OpWriteMultipleProperties,
		wot.OpSubscribeAllEvents, wot.OpUnsubscribeAllEvents,
	}
	form["contentType"] = "application/json"

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
		aff.AddForm(td.NewForm(wot.OpInvokeAction, href, SubprotocolWSS))
		aff.AddForm(td.NewForm(wot.OpQueryAction, href, SubprotocolWSS))
		// cancel action is currently not supported
		//aff.AddForm(td.NewForm(wot.OpCancelAction, href))
	}
	for name, aff := range tdoc.Events {
		_ = name
		aff.AddForm(td.NewForm(wot.OpSubscribeEvent, href, SubprotocolWSS))
		aff.AddForm(td.NewForm(wot.OpUnsubscribeEvent, href, SubprotocolWSS))
	}
	for name, aff := range tdoc.Properties {
		_ = name
		aff.AddForm(td.NewForm(wot.OpObserveProperty, href, SubprotocolWSS))
		aff.AddForm(td.NewForm(wot.OpUnobserveProperty, href, SubprotocolWSS))
		if !aff.WriteOnly {
			aff.AddForm(td.NewForm(wot.OpReadProperty, href, SubprotocolWSS))
		}
		if !aff.ReadOnly {
			aff.AddForm(td.NewForm(wot.OpWriteProperty, href, SubprotocolWSS))
		}
	}
}
