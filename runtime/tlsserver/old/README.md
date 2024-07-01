# tlsserver

This package provides a wrapper around the http TLS server and authenticates the sender of the request using Basic, Certificate or JWT.

## Status

Functional.
This has been moved over from an older project and still needs to be brought in line with the HiveOT Hub project.

## Server Usage

NewAuthenticator provides a handler that verifies provided credentials supporting multiple protocols.

* Client Certificate authentication

    The client includes a client certificate in its TLS connection that includes its clientID in the CN and role in the OU field. The certificate is signed by the Hub CA.

* BASIC authentication. See also: https://www.alexedwards.net/blog/basic-authentication-in-go
  
  Parse the Authorization header, where base64 is a function that encodes the "username:password" string in base64 format.
   > Authorization: Basic base64("username:password")

  
* [DIGEST authentication](https://www.rfc-editor.org/rfc/rfc7616.txt)
  1. Client performs GET request
  2. Server responds with 401, header: WWW-authenticate: Digest, and fields real, qop, algorithm, none and opaque.
  3. Client gets login credentials username and password from user
  4. Client repeats request including the header:
  "Authorization: Digest username="", realm=, nonce=, qop=, opaque=, algorithm=, response=, cnonce=, userhash=


* JWT authentication

  The client makes a login request providing its credentials and a requested Hash algorithm. The server returns a bearer token which is a hash of  The default hash is MD5. A different algorithm can be configured. All future request include a Authentication header with bearer token:
  > Authorization: Bearer asldkasdwerpwoierwperowepr



```golang
pwStore := unpwstore.NewPasswordFileStore(path)
httpAuthenticator := authenticator.NewHttpAuthenticator(pwStore)
router.HandleFunc(path, httpauth.NewAuthHandler(httpAuthenticator.Authenticate))
```

For JWT authentication also add a login handler to obtain a token

```golang
router.HandleFunc("/login", httpauth.LoginHandler)
```




## Client Usage

... todo .. describe authentication clients