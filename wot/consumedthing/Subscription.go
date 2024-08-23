package consumedthing

type Subscription interface {
	// Active returns the subscription active status
	Active() bool
	// Stop the Subscription
	Stop()
}
