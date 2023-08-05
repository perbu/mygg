package mygg

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

// MQTT packetType types
const (
	CONNECT     = 1
	CONNACK     = 2
	PUBLISH     = 3
	PUBACK      = 4
	PUBREC      = 5
	PUBREL      = 6
	PUBCOMP     = 7
	SUBSCRIBE   = 8
	SUBACK      = 9
	UNSUBSCRIBE = 10
	UNSUBACK    = 11
	PINGREQ     = 12
	PINGRESP    = 13
	DISCONNECT  = 14
)

// MQTTQoS is MQTT Quality of Service
type MQTTQoS byte

// QoS levels
const (
	QoS0 MQTTQoS = iota // At most once
	QoS1                // At least once
	QoS2                // Exactly once
)

// MQTTPacket is the interface for all types of MQTT packets
type MQTTPacket interface {
	PacketType() byte
	PacketID() MQTTPacketID
}

// MQTTConnectPacket is the CONNECT Packet
type MQTTConnectPacket struct {
	ClientID string
}

func (p *MQTTConnectPacket) PacketType() byte {
	return CONNECT
}

func (p *MQTTConnectPacket) PacketID() MQTTPacketID {
	return 0
}

// MQTTClient is the MQTT client
type MQTTClient struct {
	clientId string
	conn     net.Conn
	packetID MQTTPacketID
}

func New(clientID string) *MQTTClient {
	return &MQTTClient{
		clientId: clientID,
	}
}
func (c *MQTTClient) Connect(url string) error {
	ntype, address, err := parseURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	conn, err := net.Dial(ntype, address)
	if err != nil {
		return err
	}
	c.conn = conn
	packet := &MQTTConnectPacket{
		ClientID: c.clientId,
	}

	// Serialize packetType and write to connection
	// We'll implement this in the next step
	err = c.writePacket(packet)
	if err != nil {
		return err
	}

	// TODO: Raead and handle CONNACK packetType

	return nil
}

func (c *MQTTClient) Disconnect() error {
	packet := &MQTTDisconnectPacket{}
	return c.writePacket(packet)
}

// DISCONNECT Packet
type MQTTDisconnectPacket struct{}

func (p *MQTTDisconnectPacket) PacketType() byte {
	return DISCONNECT
}

func (p *MQTTDisconnectPacket) PacketID() MQTTPacketID {
	return 0
}

// parseURL parses the URL and returns the network type and address
// it takes the URL in the format of tcp://localhost:1883
func parseURL(url string) (string, string, error) {
	parts := strings.Split(url, "://")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid URL: %s", url)
	}
	ctype, addr := parts[0], parts[1]
	if ctype != "tcp" {
		return "", "", fmt.Errorf("unsupported network type: %s", ctype)
	}
	return ctype, addr, nil
}
func (c *MQTTClient) writePacket(packet MQTTPacket) error {
	switch packet.PacketType() {
	case CONNECT:
		return c.writeConnectPacket(packet.(*MQTTConnectPacket))
	case DISCONNECT:
		return c.writeDisconnectPacket(packet.(*MQTTDisconnectPacket))
	case PUBLISH:
		return c.writePublishPacket(packet.(*MQTTPublishPacket))
	case SUBSCRIBE:
		return c.writeSubscribePacket(packet.(*MQTTSubscribePacket))
	default:
		return fmt.Errorf("unsupported packetType type: %d", packet.PacketType())
	}
}

func (c *MQTTClient) writeConnectPacket(packet *MQTTConnectPacket) error {
	// Fixed header
	// Packet type and flags (fixed to 0 for CONNECT)
	header := byte(CONNECT << 4)

	// Variable header
	// Protocol name length
	protocolNameLength := make([]byte, 2)
	binary.BigEndian.PutUint16(protocolNameLength, uint16(len("MQTT")))
	// Protocol name
	protocolName := []byte("MQTT")
	// Protocol level
	protocolLevel := byte(4)
	// Connect flags
	connectFlags := byte(2) // Clean session flag is set
	// Keep alive (10 seconds)
	keepAlive := make([]byte, 2)
	binary.BigEndian.PutUint16(keepAlive, uint16(10))

	variableHeader := append(protocolNameLength, protocolName...)
	variableHeader = append(variableHeader, protocolLevel, connectFlags)
	variableHeader = append(variableHeader, keepAlive...)

	// Payload
	// Client ID length
	clientIDLength := make([]byte, 2)
	binary.BigEndian.PutUint16(clientIDLength, uint16(len(packet.ClientID)))
	// Client ID
	clientID := []byte(packet.ClientID)

	payload := append(clientIDLength, clientID...)

	// Remaining length
	remainingLength := len(variableHeader) + len(payload)
	remainingLengthEncoded := encodeRemainingLength(remainingLength)

	packetBytes := append([]byte{header}, remainingLengthEncoded...)
	packetBytes = append(packetBytes, variableHeader...)
	packetBytes = append(packetBytes, payload...)

	_, err := c.conn.Write(packetBytes)
	return err
}

func (c *MQTTClient) writeDisconnectPacket(packet *MQTTDisconnectPacket) error {
	// Fixed header
	// Packet type and flags (fixed to 0 for DISCONNECT)
	header := byte(DISCONNECT << 4)
	// Remaining length (0 for DISCONNECT)
	remainingLength := encodeRemainingLength(0)

	packetBytes := append([]byte{header}, remainingLength...)

	_, err := c.conn.Write(packetBytes)
	return err
}

// Encode the remaining length field as per MQTT spec
func encodeRemainingLength(length int) []byte {
	encoded := make([]byte, 0)

	for {
		digit := length % 128
		length = length / 128
		// if there are more digits to encode, set the top bit of this digit
		if length > 0 {
			digit = digit | 0x80
		}
		encoded = append(encoded, byte(digit))
		if length == 0 {
			break
		}
	}

	return encoded
}

// MQTTPublishPacket is the PUBLISH Packet
type MQTTPublishPacket struct {
	Topic   string
	Payload []byte
	QoS     MQTTQoS
}

func (p *MQTTPublishPacket) PacketType() byte {
	return PUBLISH
}

func (p *MQTTPublishPacket) PacketID() MQTTPacketID {
	return 0
}

func (c *MQTTClient) Publish(topic string, payload []byte) error {
	packet := &MQTTPublishPacket{
		Topic:   topic,
		Payload: payload,
		QoS:     QoS0,
	}
	return c.writePacket(packet)
}

func (c *MQTTClient) writePublishPacket(packet *MQTTPublishPacket) error {
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

// MQTTPacketID represents Packet Identifier
type MQTTPacketID uint16

// MQTTSubscribePacket is the SUBSCRIBE Packet
type MQTTSubscribePacket struct {
	packetID MQTTPacketID
	Topic    string
	QoS      MQTTQoS
}

func (p *MQTTSubscribePacket) PacketType() byte {
	return SUBSCRIBE
}

func (p *MQTTSubscribePacket) PacketID() MQTTPacketID {
	return p.packetID
}

func (c *MQTTClient) Subscribe(topic string, qos MQTTQoS) error {
	// Increment packet ID for each new packet
	c.packetID++
	packet := &MQTTSubscribePacket{
		packetID: c.packetID,
		Topic:    topic,
		QoS:      qos,
	}
	return c.writePacket(packet)
}

func (c *MQTTClient) writeSubscribePacket(packet *MQTTSubscribePacket) error {
	// Fixed header
	// Packet type and flags (fixed to 2 for SUBSCRIBE)
	header := byte(SUBSCRIBE<<4 | 2)

	// Variable header
	// Packet ID
	packetID := make([]byte, 2)
	binary.BigEndian.PutUint16(packetID, uint16(packet.PacketID()))

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
