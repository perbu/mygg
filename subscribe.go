package mygg

import "encoding/binary"

type mqttSubscribePacket struct {
	nextPacketID mqttPacketID
	Topic        string
	QoS          MQTTQoS
}

func (p *mqttSubscribePacket) packetType() byte {
	return SUBSCRIBE
}

func (p *mqttSubscribePacket) packetID() mqttPacketID {
	return p.nextPacketID
}

func (c *MQTTClient) writeSubscribePacket(packet *mqttSubscribePacket) error {
	// Fixed header
	// Packet type and flags (fixed to 2 for SUBSCRIBE)
	header := byte(SUBSCRIBE<<4 | 2)

	// Variable header
	// Packet ID
	packetID := make([]byte, 2)
	binary.BigEndian.PutUint16(packetID, uint16(packet.packetID()))

	// Payload
	// Topic name
	topicLength := make([]byte, 2)
	binary.BigEndian.PutUint16(topicLength, uint16(len(packet.Topic)))
	topic := []byte(packet.Topic)
	// Requested QoS
	qos := []byte{byte(packet.QoS)}

	payload := append(topicLength, topic...)
	payload = append(payload, qos...)

	// Remaining length
	remainingLength := 2 + len(payload) // 2 for packet ID, then payload
	remainingLengthEncoded := encodeRemainingLength(remainingLength)

	packetBytes := append([]byte{header}, remainingLengthEncoded...)
	packetBytes = append(packetBytes, packetID...)
	packetBytes = append(packetBytes, payload...)

	_, err := c.conn.Write(packetBytes)
	return err
}
