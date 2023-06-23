package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	"nhooyr.io/websocket"
)

// A nice, simple function to handle commands.
func handle(input string) (string, bool) {
	if input == "set-gpio high" {
		pin := rpio.Pin(18)
		pin.Output()
		pin.High()
		return "", false
	} else if input == "set-gpio low" {
		pin := rpio.Pin(18)
		pin.Output()
		pin.Low()
		return "", false
	} else if input == "get-gpio" {
		pin := rpio.Pin(18)
		res := pin.Read()
		return strconv.Itoa(int(res)), true
	}

	return input, true
}

func connectHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	log.Println("Connected!")

	for {
		err := read(r.Context(), c)
		if err != nil {
			return
		}

		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
	}
}

func read(ctx context.Context, c *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	typ, r, err := c.Reader(ctx)
	if err != nil {
		return err
	}

	buf := new(strings.Builder)
	n, err := io.Copy(buf, r)

	cmd := ""

	if n > 0 {
		cmd = strings.TrimSuffix(buf.String(), "\n")
		log.Printf("Received: '%s'", cmd)
	}

	// Handle our commands
	res, send := handle(cmd)

	if send {
		w, err := c.Writer(ctx, typ)
		if err != nil {
			return err
		}

		_, err = w.Write(bytes.NewBufferString(res).Bytes())

		if err != nil {
			return fmt.Errorf("failed to w.Write: %w", err)
		}
		err = w.Close()
	}
	return err
}
