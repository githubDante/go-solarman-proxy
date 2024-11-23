package client

import (
	"encoding/hex"
	"strings"
)

/*
POC
#1 ping packet
a5 0100 1047 0000 00000000 00 <V5 chksum> 15
Final:
a5 0100 1047 0000 00000000 00 58 15

#2 A read attempt with an empty packet

read_discrete_inputs
a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 02 00 01 00 01 e8 0a 65 15
read_input_registers
a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 04 00 01 00 01 60 0a df 15
read_holding_registers
a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 03 00 01 00 01 d5 ca 13 15
read_coils
a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 01 00 01 00 01 ac 0a 28 15


Notes:
#1 does not work - no response on ping/counter
#2 it works - different functions can be used
*/

type V5Payload string

const (
	ReadCoils    V5Payload = "a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 01 00 01 00 01 ac 0a 28 15"
	ReadDiscrete V5Payload = "a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 02 00 01 00 01 e8 0a 65 15"
	ReadHolding  V5Payload = "a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 03 00 01 00 01 d5 ca 13 15"
	ReadInput    V5Payload = "a5 17 00 10 45 00 00 00 00 00 00 02 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01 04 00 01 00 01 60 0a df 15"
)

func (v *V5Payload) String() string {
	switch *v {
	case ReadDiscrete:
		return "ReadDiscrete"
	case ReadInput:
		return "ReadInput"
	case ReadHolding:
		return "ReadHolding"
	case ReadCoils:
		return "ReadCoils"
	default:
		return "Unknown"
	}
}

func (v *V5Payload) ToBytes() []byte {
	s := strings.ReplaceAll(string(*v), " ", "")
	b, _ := hex.DecodeString(s)
	return b
}

func AllPayloads() []V5Payload {
	return []V5Payload{
		ReadCoils, ReadDiscrete, ReadInput, ReadHolding,
	}
}
