package mqttmsgserver

import (
	"fmt"
	"github.com/hiveot/hub/api/go/msgserver"
)

func (srv *MqttMsgServer) ApplyAuth(clients []msgserver.ClientAuthInfo) error {
	return fmt.Errorf("not yet implemented")
}

func (srv *MqttMsgServer) CreateToken(clientID string, pubKey string) (token string, err error) {
	return "", fmt.Errorf("not yet implemented")
}

func (srv *MqttMsgServer) CreateJWTToken(clientID string, pubKey string) (newToken string, err error) {
	return "", fmt.Errorf("not yet implemented")
}
func (srv *MqttMsgServer) SetRolePermissions(
	rolePerms map[string][]msgserver.RolePermission) {
	srv.rolePermissions = rolePerms
}

func (srv *MqttMsgServer) SetServicePermissions(
	serviceID string, capability string, roles []string) {
}

func (srv *MqttMsgServer) ValidateJWTToken(
	clientID string, pubKey string, tokenString string, signedNonce string, nonce string) error {
	return fmt.Errorf("not yet implemented")
}

func (srv *MqttMsgServer) ValidatePassword(loginID string, password string) error {
	return fmt.Errorf("not yet implemented")
}

func (srv *MqttMsgServer) ValidateToken(
	clientID string, pubKey string, oldToken string, signedNonce string, nonce string) (err error) {
	return fmt.Errorf("not yet implemented")
}
