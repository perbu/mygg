package mygg

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

const startMosquitto = false

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
	c := New("test")
	err := c.Connect("tcp://localhost:1883")
	if err != nil {
		t.Errorf("Connect error: %v", err)
	}
	time.Sleep(time.Millisecond * 200)
	err = c.Publish("test", []byte("test"))
	if err != nil {
		t.Errorf("Publish error: %v", err)
	}
	err = c.Subscribe("test", MQTTQoS(0))
	if err != nil {
		t.Errorf("Subscribe error: %v", err)
	}
	err = c.Disconnect()
	if err != nil {
		t.Errorf("Disconnect error: %v", err)
	}
}
