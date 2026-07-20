package websocket

// capabilityConn is the erased boundary for Go+'s indexed, linear capability.
// The Go+ checker proves which constructor is possible at each call site.
func capabilityConn(capability Capability) *Conn {
	switch value := capability.(type) {
	case OpenCapability:
		return value.Conn
	case CloseSentCapability:
		return value.Conn
	case ClosedCapability:
		return value.Conn
	default:
		panic("websocket: invalid capability")
	}
}
