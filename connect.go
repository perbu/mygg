package mygg

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

func (c *MQTTClient) Connect(ctx context.Context, url string) error {

	ntype, address, err := parseURL(url)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	conn, err := net.Dial(ntype, address)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", url, err)
	}
	c.conn = conn
	packet := &mqttConnectPacket{
		ClientID: c.clientId,
	}

	// Serialize packetType and write to connection
	err = c.writePacket(packet)
	if err != nil {
		return fmt.Errorf("write CONNECT packet: %w", err)
	}

	err = c.readConnAck()
	if err != nil {
		return fmt.Errorf("read CONNACK: %w", err)
	}
	// we're connected. start reading packets and dispatching them to the handler
	go c.readPump(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case packet := <-c.pCh:
			c.handlePacket(packet)
		}
	}
}

// mqttConnectPacket is the CONNECT Packet
type mqttConnectPacket struct {
	ClientID string
}

func (p *mqttConnectPacket) packetType() byte {
	return CONNECT
}

func (p *mqttConnectPacket) packetID() mqttPacketID {
	return 0
}

func (c *MQTTClient) readConnAck() error {
	buffer := make([]byte, 4) // CONNACK packet is 4 bytes
	n, err := io.ReadFull(c.conn, buffer)
	if err != nil {
		return fmt.Errorf("failed to read CONNACK: %w", err)
	}
	if n != 4 {
		return fmt.Errorf("failed to read CONNACK: read %d bytes, expected 4", n)
	}

	// MQTT CONNACK packet structure
	// byte 1: Packet type and flags (can be ignored)
	// byte 2: Remaining length (always 2 for CONNACK, can be ignored)
	// byte 3: Connect Acknowledge Flags (can be ignored for now)
	// byte 4: Connect Return Code
	returnCode := buffer[3]

	switch returnCode {
	case 0:
		return nil // Connection Accepted
	case 1:
		return errors.New("connection refused: unacceptable protocol version")
	case 2:
		return errors.New("connection refused: identifier rejected")
	case 3:
		return errors.New("connection refused: server unavailable")
	case 4:
		return errors.New("connection refused: bad user name or password")
	case 5:
		return errors.New("connection refused: not authorized")
	default:
		return errors.New("unknown CONNACK error")
	}
}

func (c *MQTTClient) writeConnectPacket(packet *mqttConnectPacket) error {
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
