package wavesharecomm

import (
	"io"
	"log"

	"github.com/jacobsa/go-serial/serial"
)

func OpenPort(portPath string) (io.ReadWriteCloser, error) {
	portOptions := serial.OpenOptions{
		PortName:              portPath,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		InterCharacterTimeout: 2000,
	}
	return serial.Open(portOptions)
}

func WriteCommand(port io.Writer, cmd string) error {
	newCmd := append([]byte("AT"), append([]byte(cmd), "\r\n"...)...)
	log.Printf("Sending command over serial: %v", newCmd)
	_, err := port.Write(newCmd)
	return err
}
