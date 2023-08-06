# mygg

Mygg is a MQTT library for Go. At the moment it's completely basic and not suited for actual use.

## Design goal:
 - simplicity
 - correctness

Since MQTT is stateful, the library is also stateful. The main difference from other 
implementations is that there is no automatic reconnect. The library will not try to
reconnect if the connection is lost. It's up to the user to handle this. 