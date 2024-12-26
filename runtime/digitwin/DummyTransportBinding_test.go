package digitwin_test

// dummy transport for testing with the digitwin service
// This implements the ITransportServer interface.
//type DummyTransportServer struct {
//}

//func (dummy *DummyTransportSe
//func (dummy *DummyTransportServer) GetConnectionByConnectionID(cid string) transports2.IServerConnection {
//	return nil
//}
//func (dummy *DummyTransportServer) GetProtocolType() string {
//	return transports2.ProtocolTypeEmbedded
//}

//func (dummy *DummyTransportServer) InvokeAction(
//	agentID string, thingID string, name string, value any, requestID string, consumerID string) (
//	status string, output any, err error) {
//
//	return vocab.RequestPending, nil, nil
//}
//
//func (dummy *DummyTransportServer) PublishEvent(
//	dThingID string, name string, value any, requestID string, agentID string) {
//}
//
//func (dummy *DummyTransportServer) PublishProperty(
//	dThingID string, name string, value any, requestID string, agentID string) {
//}
//func (dummy *DummyTransportServer) PublishProgressUpdate(
//	connectionID string, stat transports2.ActionStatus, agentID string) (bool, error) {
//	return false, nil
//}
//
//func (dummy *DummyTransportServer) WriteProperty(
//	agentID string, thingID string, name string, value any, msgID string, senderID string) (
//	found bool, status string, err error) {
//
//	return false, vocab.RequestPending, nil
//}

//func NewDummyTransportServer() *DummyTransportServer {
//	dummy := DummyTransportServer{}
//	return &dummy
//}
