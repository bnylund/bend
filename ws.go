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
	command, ok := getArgument(input, 0)
	if !ok {
		return "", false
	}

	// set-gpio 18 output high
	if command == "set-gpio" {
		pinNum, ok := getArgument(input, 1)
		if !ok {
			return "", false
		}
		pinNumber, err := strconv.Atoi(pinNum)
		if err != nil {
			return "", false
		}
		mode, ok := getArgument(input, 2)
		if !ok {
			return "", false
		}
		pin := rpio.Pin(pinNumber)
		if mode == "output" {
			pin.Output()
		} else if mode == "input" {
			pin.Input()
		}
		value, ok := getArgument(input, 3)
		if !ok {
			return "", false
		}
		if value == "1" {
			pin.High()
			return input, true
		} else if value == "0" {
			pin.Low()
			return input, true
		}
		return "", false
	} else if command == "get-gpio" {
		pinNum, ok := getArgument(input, 1)
		if !ok {
			return "", false
		}
		pinNumber, err := strconv.Atoi(pinNum)
		if err != nil {
			return "", false
		}
		pin := rpio.Pin(pinNumber)
		res := pin.Read()
		return strconv.Itoa(int(res)), true
	} else if command == "subscribe" {
		pinNum, ok := getArgument(input, 1)
		if !ok {
			return "", false
		}
		pinNumber, err := strconv.Atoi(pinNum)
		if err != nil {
			return "", false
		}
		pin := rpio.Pin(pinNumber)
		pin.Detect(rpio.RiseEdge)
		edgePins = append(edgePins, pinNumber)
		return "", false
	} else if input == "ping" {
		return "pong", true
	}

	return input, true
}

func connectHandler(pool *pool, w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})

	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	conn := connection{
		ws:   c,
		pool: pool,
	}

	pool.connections[&conn] = true

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
		if cmd != "ping" {
			log.Printf("Received: '%s'", cmd)
		}
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
		w.Close()
	}
	return err
}

func writeToConn(conn *websocket.Conn, input string) error {
	context := context.Background()
	w, err := conn.Writer(context, websocket.MessageText)
	if err != nil {
		return err
	}

	_, err = w.Write(bytes.NewBufferString(input).Bytes())

	if err != nil {
		return fmt.Errorf("failed to w.Write: %w", err)
	}
	return w.Close()
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// The connection pool.
	pool *pool
}

type pool struct {
	// Registered connections. That's a connection pool
	connections map[*connection]bool
}
