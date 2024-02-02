# Session Management

Hiveoview is designed with SSR (Server Side Rendering) and stateful session management on the server.

## Session Cookie

The hiveoview server does not persist session information. A server restart results in removing all active sessions from memory. To allow for an easy reconnect, cookies are used to store session information.

For security, cookies are stored with the http-only and same-site=strict flags set. This reduces the risk of XSS and CSRF attacks.

In addition, cookie content is encrypted by the server. Stealing a cookie will not expose the login credentials.

The data stored in a cookie is a server encrypted JWT token containing claims for:

1. the sessionID, used to differentiate sessions from different browsers.
2. the loginID, used to reconnect a session without having to login again.
3. authentication token (short lived) of the Hub connection, used to restore a Hub connection without requiring a password.
4. remote IP address the session belongs to, used to prevent use of stolen cookies.
5. browser ID the session belongs to, also used to prevent use of stolen cookies.

Note that the session cookie can still be viewed by the user of the browser. On public computers it is important to set the max-age of the cookie to 'session' so it is removed when the browser closes. In the UI an option 'remember me' at the login screen controls the max-age settings.

## Session Login

On a login request, the backend first verifies the credentials with the Hub.
It does this by connecting a Hub client using the given credentials.

If the connection fails, the user is directed back to the login page with an error message. If a valid session already exists it will remain unaffected.

If the connection is successful, a session is requested from the session manager using the connected Hub client. There are 3 cases to deal with:

1. A session already exists and is disconnected. In this case the new Hub connection replaces the disconnected connection and the existing session is kept.
2. A session already exists and has a connected Hub client. The existing session and connection are kept as to not lose existing subscriptions and SSE connections. The provided client is discarded. This achieves that the auth token has been refreshed in the cookie.
3. A session doesn't exist. A new session is created using the given Hub client and session ID.

After the session is activated the session cookie is updated.

## Session Validation

Requests to the backend are first checked for an active session by the session-authentication middleware. An active session is one with a connection to the Hub. The session is identified the clientID in the session cookie. If an active session exists then the session is added to the request context and passed down the router chain for further handling.

If no valid cookie is present then the user is considered unauthenticated. The client is redirected to the login page along with a message 'please remind us who you are.'.

If a session cookie exists but no session is established then the session activation flow is followed as described below.

## Session Reactivation

If an inactive session exists, an attempt is made to connect to the Hub using the
clientID and authentication token from the cookie.

If the connection succeeds then a refreshed authentication token is obtained and the cookie is updated. The session is activated and passed down the chain in the request context.

If connection fails then the Connection Failure flow is followed.

## Connection Failure

If the connection fails with an unauthenticated error, the session is considered expired. The session and session cookie are removed and the request is redirected to the login page with the message 'please remind us who you are.'.

If the connection fails due to another error, eg server unavailable or internal server error, then the session is considered valid (for now) but remains inactive. The request is redirected to an error page with the message that the server is unreachable, please try again later. A cause field shows the error message.

## Login

On login, the client submits a form containing the login ID and password. The server side attempts to connect to the Hub using these credentials. On success, an authentication token is obtained and stored in a new session cookie along with the client ID. The server side creates a session instance containing the Hub connection. At this point the client is considered to be authenticated and connected.

If connection doesn't succeed then the Connection Failure flow described above is followed.

## Once Connected

Once connected, all further requests contain the session in the request context. The backend uses the session for access the Hub as requested.

If a request to the Hub unexpectedly fails while serving the request, an error is returned.

## Session Deactivation

Session Deactivation is the process of disconnecting an active session from the Hub without losing authentication. Deactivation is initiated after the session has not been used for a period of time to reduce unused resources.

The session manager periodically checks the last activity of each active session and deactivates them as needed.

Successive requests will automatically reactive the session without interruption to the client. See the session reactivation flow described above.

Sessions with an active SSE (Serve side event) connection will not be deactivated. See SSE below

## SSE Connections

A server-side-event connection follows the same flow as any other request through the middleware, except they don't return until much later. The connection is used to send events to the client, eliminating the need for polling.

The session stores active SSE connections. As long as one or more SSE connections exist, the session will not be deactivated.

Each SSE connection registers itself with the session, providing a go channel it is listening on. The SSE listens for messages on the channel and forwards them to the client. If writing to the SSE connection fails, then the SSE connection is considered closed and removed from the session.

Removing an SSE connection from the session is considered activity and will delay deactivating the session. A browser reload will disconnect its SSE connection but the connection will be reestablished quickly. In this case there is no need to deactivate the session.

Multiple SSE connections are supported per session. Note that an event send to a session will be send to all SSE connections (fan-out). This means that if the application is open on multiple tabs, each tab will receive the same event.

## Multiple Accounts (TBD)

Using multiple accounts in the same browser is currently not supported as there is no good use-case that warrants the extra complexity.

To support multiple accounts in the same browser it could be done as follows:

After login the application URL contains an account prefix {prefix}:
> https://myhiveot-address/{prefix}/{page}

Where {prefix} is an arbitrary string and {page} is the page to view.

The URL with prefix is also used to load HTMX fragments on:

> https://myhiveot-address/{prefix}/htmx/{fragment}

The login URL is not associated with an account and has the path "/login".

> https://myhiveot-address/login

At login, the server assigns a prefix based on the cookies. For example "/u/0" (this is what Google does). The server scans the existing cookies and checks if one matching the loginID exists. If it does its prefix is used, a new stateful server session started and the browser is redirected to use the prefix path.
session started using the cookie. If the loginID isn't found then the first available prefix is used and a new cookie stored with the prefix path.

The next login gets the next prefix "/u/1". Each login adds a secure cookie containing the prefix as a path.

Server side, only a single session cookie will be found for that prefix. A different account will have a different prefix path and thus a different session cookie and thus a different userID and auth token.


