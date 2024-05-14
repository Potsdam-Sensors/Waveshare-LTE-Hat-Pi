package wavesharecomm

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

const (
	CmdStart           = ""
	CmdNoEcho          = "E0"
	CmdEcho            = "E1"
	CmdNetworkOperator = "COPS"
	portPath           = "/dev/ttyUSB2"
)

var OkResponseOk = []byte{'O', 'K'}

/*
Get the port object for communicating with the hat,
open with the correct configuration.

Please defer `.Close()` on the `port` in the calling function.
*/
func OpenWaveshareHatSerialPort() (port io.ReadWriteCloser, err error) {
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
	log.Printf("Sending command over serial: %s", newCmd)
	_, err := port.Write(newCmd)
	return err
}

/*
Read a line from a Bufio Reader, ignoring io.EOF.

Context passed should have a timeout or some other method of cancelation to prevent blocking forever
*/
func readLineIgnoreEof(ctx context.Context, reader *bufio.Reader) (serialLine []byte, err error) {
	for {
		select {
		case <-ctx.Done():
			return []byte{}, errors.New("operation timed out")
		default:
			b, _, err := reader.ReadLine()
			if err == io.EOF {
				continue
			}
			return b, err
		}
	}
}

/*
[Goroutine]

Read a response from serial on the given `reader`.
The response, if there is one, is put onto the `responseCh`.

Will also monitor the context to see if it is cancelled.
On exit, the goroutine will close its `responseCh`.
*/
func goWaitForResponse(reader *bufio.Reader, ctx context.Context, responseCh chan []byte) {
	defer close(responseCh)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Format of responses is <CR><LF> RESPONSE <CR><LF>
			// So we will read a line to remove the first <CR><LF>,
			// 		then read until the next one to get the response.
			if _, err := readLineIgnoreEof(ctx, reader); err != nil {
				log.Printf("error reading serial: %v", err)
				return
			}
			line, err := readLineIgnoreEof(ctx, reader)
			if err != nil {
				log.Printf("error reading serial: %v", err)
			}

			responseCh <- line
			return
		}
	}
}

/*
[Blocking]

Wait for a response from the serial connected to the given `reader`,
or timeout after the given `timeoutDuration`.

Returns the response, if there is one, and an error if one occurs.
*/
func waitForResponse(reader *bufio.Reader, timeoutDuration time.Duration) (response []byte, err error) {
	// Create a timeout context for the given timeout duration and a channel for the response from serial
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	responseCh := make(chan []byte, 1)

	// Wait for a response from serial
	// The goroutine will exit on any error, including context timeout and close its response channel on the way out
	go goWaitForResponse(reader, ctx, responseCh)
	select {
	case <-ctx.Done():
		return []byte{}, errors.New("operation timed out")
	case response, ok := <-responseCh:
		if !ok {
			return []byte{}, errors.New("channel closed unexpectedly")
		}
		return response, nil
	}
}

/*
[Blocking]

Run a given command, `cmd` on the given port,
returning the response (`response`) from the device and whether OK or ERROR was returned (`ok`).

Times out if the given `timeDuration` happens before the response from the device.
*/
func ExecuteCommand(port io.ReadWriter, cmd string, timeoutDuration time.Duration) (response []byte, ok bool, err error) {
	// First send the command to the hat
	err = WriteCommand(port, cmd)
	if err != nil {
		return response, false, fmt.Errorf("error writing command to port: %w", err)
	}

	// Construct a reader for all read operations
	reader := bufio.NewReader(port)

	// First try to get a response from serial
	response, err = waitForResponse(reader, timeoutDuration)
	if err != nil {
		return response, false, fmt.Errorf("error getting response from port: %w", err)
	}

	// If there was a response, and it the first char is +, then it was a response to a command
	// 		so, we need to read in the OK/ERROR line
	// Then we need to see if okResponse is "OK", or response if there was no "+" response
	if (len(response) >= 1) && (response[0] == byte('+')) {
		okResponse, err := waitForResponse(reader, timeoutDuration)
		if err != nil {
			return response, false, fmt.Errorf("error getting ok response from port: %w", err)
		}
		// log.Printf("OK?: %s", okResponse)
		ok = bytes.Equal(bytes.TrimSpace(okResponse), OkResponseOk)
	} else {
		ok = bytes.Equal(bytes.TrimSpace(response), OkResponseOk)
	}
	return response, ok, nil
}

/*
[Blocking]

Run a given command, `cmd` on the given port after formatting as a read command (+%s?),
returning the response (`response`) from the device and whether OK or ERROR was returned (`ok`).

Times out if the given `timeDuration` happens before the response from the device.
*/
func ExecuteCommandRead(port io.ReadWriter, cmd string, timeoutDuration time.Duration) ([]byte, bool, error) {
	return ExecuteCommand(port, fmt.Sprintf("+%s?", cmd), timeoutDuration)
}

/*
[Blocking]

Run a given command, `cmd` on the given port after formatting as a execute command (+%s),
returning the response (`response`) from the device and whether OK or ERROR was returned (`ok`).

Times out if the given `timeDuration` happens before the response from the device.
*/
func ExecuteCommandExecute(port io.ReadWriter, cmd string, timeoutDuration time.Duration) ([]byte, bool, error) {
	return ExecuteCommand(port, fmt.Sprintf("+%s", cmd), timeoutDuration)
}
