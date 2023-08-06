package mygg

func New(clientID string) *MQTTClient {
	return &MQTTClient{
		clientId: clientID,
	}
}

func (c *MQTTClient) Disconnect() error {
	packet := &mqttDisconnectPacket{}
	return c.writePacket(packet)
}

func (c *MQTTClient) Subscribe(topic string, qos MQTTQoS) error {
	// Increment packet ID for each new packet
	id := c.packetID.Add(1)
	packet := &mqttSubscribePacket{
		nextPacketID: mqttPacketID(id),
		Topic:        topic,
		QoS:          qos,
	}
	return c.writePacket(packet)
}

func (c *MQTTClient) Publish(topic string, payload []byte) error {
	packet := &mqttPublishPacket{
		Topic:   topic,
		Payload: payload,
		QoS:     QoS0,
	}
	return c.writePacket(packet)
}
