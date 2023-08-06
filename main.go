package mygg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
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

// mqttPacket is the interface for all types of MQTT packets
type mqttPacket interface {
	packetType() byte
	packetID() mqttPacketID
}

// MQTTClient is the MQTT client
type MQTTClient struct {
	clientId string
	conn     net.Conn
	packetID atomic.Uint32
	mu       sync.Mutex
	pCh      chan mqttPacket
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

func (c *MQTTClient) writePacket(packet mqttPacket) error {
	switch packet.packetType() {
	case CONNECT:
		return c.writeConnectPacket(packet.(*mqttConnectPacket))
	case DISCONNECT:
		return c.writeDisconnectPacket(packet.(*mqttDisconnectPacket))
	case PUBLISH:
		return c.writePublishPacket(packet.(*mqttPublishPacket))
	case SUBSCRIBE:
		return c.writeSubscribePacket(packet.(*mqttSubscribePacket))
	default:
		return fmt.Errorf("unsupported packetType type: %d", packet.packetType())
	}
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

// mqttPacketID represents Packet Identifier
type mqttPacketID uint16

// mqttSubscribePacket is the SUBSCRIBE Packet

// readPump reads packets from the connection and sends them into the packet channel
func (c *MQTTClient) readPump(ctx context.Context) {
	for {
		packet, err := c.readPacket(ctx)
		if err != nil {
			return
		}
		c.pCh <- packet
	}
}

func (c *MQTTClient) readPacket(ctx context.Context) (mqttPacket, error) {
	// Fixed header
	// First byte is packet type and flags
	header := make([]byte, 1)
	_, err := io.ReadFull(c.conn, header)
	if err != nil {
		return nil, fmt.Errorf("read packet header: %w", err)
	}
	packetType := header[0] >> 4
	flags := header[0] & 0x0F

	// Remaining length (variable-length encoding)
	remainingLength, err := c.readRemainingLength()
	if err != nil {
		return nil, fmt.Errorf("read remaining length: %w", err)
	}

	// Now we need to read the rest of the packet based on its type
	switch packetType {
	case CONNACK:
		return c.readConnAckPacket(remainingLength)
	case PUBLISH:
		return c.readPublishPacket(flags, remainingLength)
	default:
		// TODO: Implement for other packet types
		return nil, fmt.Errorf("unsupported packet type %d", packetType)
	}
}

type mqttConnAckPacket struct {
	ReturnCode byte
}

func (m mqttConnAckPacket) packetType() byte {
	return CONNACK
}
func (m mqttConnAckPacket) packetID() mqttPacketID {
	return 0
}

func (c *MQTTClient) readConnAckPacket(remainingLength int) (*mqttConnAckPacket, error) {
	// Ensure the remaining length is 2 for a valid CONNACK packet
	if remainingLength != 2 {
		return nil, errors.New("invalid CONNACK packet: incorrect remaining length")
	}

	buffer := make([]byte, 2)
	_, err := io.ReadFull(c.conn, buffer)
	if err != nil {
		return nil, err
	}

	// Ignore the first byte (connect acknowledge flags)
	returnCode := buffer[1]

	return &mqttConnAckPacket{ReturnCode: returnCode}, nil
}

func (c *MQTTClient) readRemainingLength() (int, error) {
	multiplier := 1
	value := 0
	for {
		encodedByte := make([]byte, 1)
		_, err := io.ReadFull(c.conn, encodedByte)
		if err != nil {
			return 0, err
		}
		value += int(encodedByte[0]&127) * multiplier
		if encodedByte[0]&128 == 0 {
			break
		}
		multiplier *= 128
		if multiplier > 128*128*128 {
			return 0, errors.New("malformed remaining length")
		}
	}
	return value, nil
}

func (c *MQTTClient) handlePacket(packet mqttPacket) error {
	switch packet.packetType() {
	default:
		return fmt.Errorf("unsupported packet type %d", packet.packetType())
	}
}
