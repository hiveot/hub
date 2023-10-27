/** Account connection status definition
 */
export interface IConnectionStatus {
	// Client ID when authenticated
	clientID: string;

	// human description of authentication status
	// authStatusMessage: string;

	// the directory is obtained
	// directory: boolean;

	// gatewayclient connection is established
	status: 'connected' | 'authenticated' | 'disconnected';

	// human description of connection status
	statusMessage: string;
}
