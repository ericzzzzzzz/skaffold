package main

import (
	"io"
	"log"
	"net"
	"os"
)

func main() {

	done := make(chan struct{})
	conn, err := net.Dial("unix", "/tmp/downstream.sock")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	go func() {
		_, err := io.Copy(conn, os.Stdin)
		if err != nil {
			return
		}
	}()

	go func() {
		_, err := io.Copy(os.Stdout, conn)
		if err != nil {
			return
		}
	}()

	<-done
}
