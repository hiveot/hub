package capserializer

import (
	"capnproto.org/go/capnp/v3"
	"github.com/hiveot/hub/pkg/provisioning"

	"github.com/hiveot/hub/api/go/hubapi"
)

// UnmarshalProvStatus deserializes ProvisionStatus object from a capnp message
func UnmarshalProvStatus(statusCapnp hubapi.ProvisionStatus) provisioning.ProvisionStatus {
	// errors are ignored. If these fails then there are bigger problems
	statusPOGS := provisioning.ProvisionStatus{}
	statusPOGS.DeviceID, _ = statusCapnp.DeviceID()
	statusPOGS.RequestTime, _ = statusCapnp.RequestTime()
	statusPOGS.RetrySec = int(statusCapnp.RetrySec())
	statusPOGS.Pending = statusCapnp.Pending()
	statusPOGS.ClientCertPEM, _ = statusCapnp.ClientCertPEM()
	statusPOGS.CaCertPEM, _ = statusCapnp.CaCertPEM()
	return statusPOGS
}

// MarshalProvStatus serializes ProvisionStatus object to a capnp message
func MarshalProvStatus(statusPOGS provisioning.ProvisionStatus) hubapi.ProvisionStatus {
	// errors are ignored. If these fail then there are bigger problems
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	statusCapnp, _ := hubapi.NewProvisionStatus(seg)

	_ = statusCapnp.SetDeviceID(statusPOGS.DeviceID)
	_ = statusCapnp.SetRequestTime(statusPOGS.RequestTime)
	statusCapnp.SetRetrySec(int32(statusPOGS.RetrySec))
	statusCapnp.SetPending(statusPOGS.Pending)
	_ = statusCapnp.SetClientCertPEM(statusPOGS.ClientCertPEM)
	_ = statusCapnp.SetCaCertPEM(statusPOGS.CaCertPEM)
	return statusCapnp
}

// UnmarshalProvStatusList deserializes ProvisionStatus list from a capnp message
func UnmarshalProvStatusList(statusListCapnp hubapi.ProvisionStatus_List) []provisioning.ProvisionStatus {
	// errors are ignored. If these fails then there are bigger problems
	statusListPOGS := make([]provisioning.ProvisionStatus, statusListCapnp.Len())
	for i := 0; i < statusListCapnp.Len(); i++ {
		statusCapnp := statusListCapnp.At(i)
		statusPOGS := UnmarshalProvStatus(statusCapnp)
		statusListPOGS[i] = statusPOGS
	}
	return statusListPOGS
}

// MarshalProvStatusList serializes a ProvisionStatus list to a capnp message
func MarshalProvStatusList(statusListPOGS []provisioning.ProvisionStatus) hubapi.ProvisionStatus_List {
	// errors are ignored. If these fail then there are bigger problems
	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	statusListCapnp, _ := hubapi.NewProvisionStatus_List(seg, int32(len(statusListPOGS)))
	for i, statusPOGS := range statusListPOGS {
		statusCapnp := MarshalProvStatus(statusPOGS)
		statusListCapnp.Set(i, statusCapnp)
	}
	return statusListCapnp
}
