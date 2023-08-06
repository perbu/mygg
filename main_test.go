package mygg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

const startMosquitto = true

func TestMain(m *testing.M) {
	// setup
	// run mosquitto so we have a simple broker to test against
	if startMosquitto {
		cmd := exec.Command("mosquitto")
		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		defer cmd.Process.Kill() // nolint: errcheck
	}
	time.Sleep(time.Millisecond * 200)
	ret := m.Run()
	os.Exit(ret)
}

func TestMQTTClient_TestLifeCycle(t *testing.T) {
	fmt.Println("TestMQTTClient_Connect")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	c := New("test")
	err := c.Connect(ctx, "tcp://localhost:1883")
	if err != nil {
		t.Errorf("Connect error: %v", err)
	}
	fmt.Println(" - connected")
	time.Sleep(time.Millisecond * 200)
	err = c.Publish("test", []byte("test"))
	if err != nil {
		t.Errorf("Publish error: %v", err)
	}
	err = c.Subscribe("test/", MQTTQoS(0))
	if err != nil {
		t.Errorf("Subscribe error: %v", err)
	}
	fmt.Println(" - subscribed")
	err = c.Disconnect()
	if err != nil {
		t.Errorf("Disconnect error: %v", err)
	}
	fmt.Println(" - disconnected")
	cancel()
}
