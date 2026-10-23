package internal

// // PublishTDs reads and publishes the TD document of the binding,
// // gateway and its nodes.
// func (svc *IsyBinding) PublishTDs() (err error) {
// 	slog.Info("PublishTDs Gateway and Nodes")

// 	tdoc := svc.MakeBindingTD()
// 	err = svc.PublishTD(tdoc)
// 	if err != nil {
// 		err = fmt.Errorf("failed publishing binding TD: %w", err)
// 		slog.Error(err.Error())
// 		return err
// 	}

// 	if !svc.isyAPI.IsConnected() {
// 		return errors.New("not connected to the gateway")
// 	}

// 	// use the binding's agent to write TDs (why exactly?)
// 	err = svc.IsyGW.PubTD(svc.Agent)
// 	if err == nil {
// 		// read and publish the node TDs
// 		err = svc.IsyGW.ReadIsyThings()
// 		if err != nil {
// 			err = fmt.Errorf("failed reading ISY nodes from gateway: %w", err)
// 			slog.Error(err.Error())
// 			return err
// 		}
// 		//things := svc.IsyGW.ReadThings()
// 		for _, thing := range svc.IsyGW.GetIsyThings() {
// 			tdoc = thing.MakeTD()
// 			err = svc.PublishTD(tdoc)
// 			if err != nil {
// 				slog.Error("failed publishing Thing TD",
// 					"thingID", tdoc.ID, "err", err.Error())
// 			}
// 		}
// 	}
// 	return err
// }
