package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

func main() {
	log.SetFlags(0)

	// Only open GPIO if running on a Raspberry Pi
	if _, err := os.Stat("/sys/firmware/devicetree/base/serial-number"); err == nil {
		err := rpio.Open()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("Not running on a Raspberry Pi. Expect errors when accessing GPIO methods.")
	}

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("please provide an address to listen on as the first argument (ex. :9000, localhost:9000)")
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}
	log.Printf("listening on http://%v", l.Addr())

	http.HandleFunc("/api", apiHandler)
	http.HandleFunc("/ws", connectHandler)

	s := &http.Server{
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}
