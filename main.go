package main

import (
	"log"
	"time"

	"github.com/Potsdam-Sensors/waveshare-lte-hat-pi/wavesharecomm"
)

func main() {
	port, err := wavesharecomm.OpenWaveshareHatSerialPort()
	if err != nil {
		log.Fatalf("failed to open port: %v", err)
	}
	defer port.Close()

	resp, ok, err := wavesharecomm.ExecuteCommand(port, wavesharecomm.CmdStart, time.Second)
	log.Printf("Response: %v (%s)", resp, resp)
	log.Printf("OK?: %v", ok)
	log.Printf("Err: %v", err)

	resp, ok, err = wavesharecomm.ExecuteCommandRead(port, wavesharecomm.CmdNetworkOperator, time.Second)
	log.Printf("Response: %v (%s)", resp, resp)
	log.Printf("OK?: %v", ok)
	if err != nil {
		log.Fatalf("Error executing read command %s: %v", wavesharecomm.CmdNetworkOperator, err)
	}
}
