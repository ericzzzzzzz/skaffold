package main

import (
	"io"
	"net"
	"os"
)

func main() {

	sockPath := "/tmp/downstream.sock"

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	done := make(chan struct{})

	go func() {
		_, err = io.Copy(os.Stdout, conn)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		_, err = io.Copy(conn, os.Stdin)
		if err != nil {
			panic(err)
		}
	}()
	<-done
}
