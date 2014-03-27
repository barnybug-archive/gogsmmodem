package gogsmmodem

import "time"

type Packet interface{}

// +ZPASR
type ServiceStatus struct {
	Status string
}

// +ZDONR
type NetworkStatus struct {
	Network string
}

// +CMTI
type MessageNotification struct {
	Storage string
	Index   int
}

// +CSCA
type SMSCAddress struct {
	Args []interface{}
}

// +CMGR
type Message struct {
	Status    string
	Telephone string
	Timestamp time.Time
	Body      string
}

// Simple OK response
type OK struct{}

// Simple ERROR response
type ERROR struct{}

// Unknown
type UnknownPacket struct {
	Command string
	Args    []interface{}
}
