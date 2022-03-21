package main

// COMMAND
const (
	CONNECT = 1 + iota
	BIND    = 1 + iota
)

// ADDRTYPE
const (
	IPV4   = 1 + iota
	DOMAIN = 2 + iota
)

// REQUEST_STATUS
const (
	SUCCESS                = iota
	SERVER_FAILURE         = iota
	NETWORK_UNREACHABLE    = iota
	COMMAND_NOT_SUPPORTED  = iota
	ADDRTYPE_NOT_SUPPORTED = iota
)
