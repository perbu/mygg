package mygg

import (
	"encoding/binary"
	"io"
)

// mqttPublishPacket is the PUBLISH Packet
type mqttPublishPacket struct {
	Topic   string
	Payload []byte
	QoS     MQTTQoS
}

func (p *mqttPublishPacket) packetType() byte {
	return PUBLISH
}

func (p *mqttPublishPacket) packetID() mqttPacketID {
	return 0
}

func (c *MQTTClient) writePublishPacket(packet *mqttPublishPacket) error {
	// Fixed header
	// Packet type, flags (fixed to 0 for QoS 0)
	header := byte(PUBLISH << 4)

	// Variable header
	// Topic name
	topicLength := make([]byte, 2)
	binary.BigEndian.PutUint16(topicLength, uint16(len(packet.Topic)))
	topic := []byte(packet.Topic)

	variableHeader := append(topicLength, topic...)

	// Payload
	payload := packet.Payload

	// Remaining length
	remainingLength := len(variableHeader) + len(payload)
	remainingLengthEncoded := encodeRemainingLength(remainingLength)

	packetBytes := append([]byte{header}, remainingLengthEncoded...)
	packetBytes = append(packetBytes, variableHeader...)
	packetBytes = append(packetBytes, payload...)

	_, err := c.conn.Write(packetBytes)
	return err
}

func (c *MQTTClient) readPublishPacket(flags byte, remainingLength int) (*mqttPublishPacket, error) {
	// TODO: Implement properly according to MQTT spec
	// For simplicity, let's assume the topic name is less than 127 bytes

	topicLength := make([]byte, 2)
	_, err := io.ReadFull(c.conn, topicLength)
	if err != nil {
		return nil, err
	}
	topicLengthUint := binary.BigEndian.Uint16(topicLength)

	topic := make([]byte, topicLengthUint)
	_, err = io.ReadFull(c.conn, topic)
	if err != nil {
		return nil, err
	}

	// The rest of the packet is the message
	messageLength := remainingLength - 2 - int(topicLengthUint) // 2 for topic length
	message := make([]byte, messageLength)
	_, err = io.ReadFull(c.conn, message)
	if err != nil {
		return nil, err
	}

	return &mqttPublishPacket{
		Topic:   string(topic),
		Payload: message,
		// TODO: Parse QoS from flags
	}, nil
}
