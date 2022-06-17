package message

import (
	"encoding/gob"
	"net"
)

const (
	
	// traffic data
	MessageTypePacket = 0
	// routing info
	MessageTypeRouting = 1
	// routing 
	RoutingRegister = 0
	RoutingRegisterOK = 1
	RoutingRegisterFailed = 2
)

// wire message
type Message struct {
	Type int   		
	Payload interface{}
}

// packet of network traffic
type Packet struct {
	Src net.IP
	Dst net.IP
	Data []byte
}


// routing entry 
type RoutingEntry struct {
	// network
	Network net.IPNet
	// metric
	Metric int
}

// routing register msg
type Routing struct {
	// type
	Type int
	// peer's provided routings
	Routings []RoutingEntry
	// message
	Message string
}


func init() {
	gob.RegisterName("wire.Message", Message{})
	gob.RegisterName("wire.Packet",  Packet{})
	gob.RegisterName("wire.Routing", Routing{})
}