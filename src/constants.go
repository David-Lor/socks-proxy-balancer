package main

const (
	CommandConnect = 1 + iota
)

const (
	AddrTypeIPv4   = 1 + iota
	AddrTypeDomain = 2 + iota
)

const (
	RequestStatusSuccess              = iota
	RequestStatusServerFailure        = iota
	RequestStatusNetworkUnreachable   = iota
	RequestStatusCommandNotSupported  = iota
	RequestStatusAddrtypeNotSupported = iota
)
