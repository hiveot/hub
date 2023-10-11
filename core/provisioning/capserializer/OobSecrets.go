package capserializer

import (
	"capnproto.org/go/capnp/v3"
	"github.com/hiveot/hub/pkg/provisioning"

	"github.com/hiveot/hub/api/go/hubapi"
)

// MarshalOobSecrets serializes a list of OOB secrets to a Capnp message
func MarshalOobSecrets(secrets []provisioning.OOBSecret) hubapi.OOBSecret_List {

	_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
	secretListCapnp, _ := hubapi.NewOOBSecret_List(seg, int32(len(secrets)))
	for i, secret := range secrets {
		_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
		secretCapnp, _ := hubapi.NewOOBSecret(seg)
		_ = secretCapnp.SetDeviceID(secret.DeviceID)
		_ = secretCapnp.SetOobSecret(secret.OobSecret)
		_ = secretListCapnp.Set(i, secretCapnp)
	}

	return secretListCapnp
}

// UnmarshalOobSecrets deserializes a list of OOB secrets from a capnp message
func UnmarshalOobSecrets(secretListCapnp hubapi.OOBSecret_List) []provisioning.OOBSecret {
	secretListPOGS := make([]provisioning.OOBSecret, secretListCapnp.Len())
	for i := 0; i < secretListCapnp.Len(); i++ {
		secretCapnp := secretListCapnp.At(i)
		deviceID, _ := secretCapnp.DeviceID()
		oobSecret, _ := secretCapnp.OobSecret()
		secret := provisioning.OOBSecret{
			DeviceID:  deviceID,
			OobSecret: oobSecret,
		}
		secretListPOGS[i] = secret
	}

	return secretListPOGS
}
