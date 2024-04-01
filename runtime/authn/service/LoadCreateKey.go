package service

// LoadCreateUserKey loads a user's key from file.
// If the key doesn't exist then create and save a new key.
//
// This supports both ecdsa and nats nkeys (edd25519)
//func (svc *AuthManageClients) LoadCreateUserKey(keyFile string) (serializedKP string, pubKey string, err error) {
//	keyData, err := os.ReadFile(keyFile)
//	if err != nil {
//		// Create a new key. The server dictates the format.
//		serializedKP, pubKey = svc.msgServer.CreateKeyPair()
//		//ecdsaKey, success := key.(*ecdsa.PrivateKey)
//		//if success {
//		// save the key
//		err = os.WriteFile(keyFile, []byte(serializedKP), 0400)
//		return serializedKP, pubKey, err
//	}
//	// this is an existing key. Try parsing it with ecdsa and nkey formats
//	// Is this an ECDSA key?
//	ecdsaKey, err := certs.PrivateKeyFromPEM(string(keyData))
//	if err == nil {
//		pubKeyData, err := x509.MarshalPKIXPublicKey(&ecdsaKey.PublicKey)
//		if err == nil {
//			pubKey = base64.StdEncoding.EncodeToString(pubKeyData)
//		}
//		// if err then the existing public key cannot be serialized.. odd
//		return string(keyData), pubKey, err
//	}
//	// Is this an nkey?
//	userKP, err := nkeys.ParseDecoratedNKey(keyData)
//	if err == nil {
//		pubKey, err = userKP.PublicKey()
//		return string(keyData), pubKey, err
//	}
//	// unknown format
//	err = fmt.Errorf("unknown format for key in file '" + keyFile + "'")
//	return "", "", err
//}

// LoadCreateUserToken loads or creates an auth token
//
// This supports both standard jwt and nats jwt tokens
//func (svc *AuthManageClients) LoadCreateUserToken(clientID, tokenFile string) (token string, err error) {
//	tokenData, err := os.ReadFile(tokenFile)
//	if err != nil {
//		// Create a new token. The server dictates the format.
//		profile, err := svc.store.GetProfile(clientID)
//		if err != nil {
//			return "", err
//		}
//		token, err = svc.msgServer.CreateToken(msgserver.ClientAuthInfo{
//			ClientID:     profile.ClientID,
//			ClientType:   profile.ClientType,
//			PubKey:       profile.PubKey,
//			PasswordHash: "",
//			Role:         profile.Role,
//		})
//		if err == nil && tokenFile != "" {
//			err = os.WriteFile(tokenFile, []byte(token), 0400)
//		}
//		return token, err
//	}
//	return string(tokenData), nil
//}
