package mygg

// DISCONNECT Packet
type mqttDisconnectPacket struct{}

func (p mqttDisconnectPacket) packetType() byte {
	return DISCONNECT
}

func (p mqttDisconnectPacket) packetID() mqttPacketID {
	return 0
}
func (c *MQTTClient) writeDisconnectPacket(_ *mqttDisconnectPacket) error {
	// Fixed header
	// Packet type and flags (fixed to 0 for DISCONNECT)
	header := byte(DISCONNECT << 4)
	// Remaining length (0 for DISCONNECT)
	remainingLength := encodeRemainingLength(0)

	packetBytes := append([]byte{header}, remainingLength...)

	_, err := c.conn.Write(packetBytes)
	return err
}
