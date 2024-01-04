/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	"context"
	pb "downstream-sync/filedownload"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	//pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	defaultName = "world"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

type RR struct {
	io.Reader
	io.Writer
}

func (r *RR) Close() error {
	return nil
}

func (r *RR) LocalAddr() net.Addr {
	return &net.UnixAddr{
		Name: "",
		Net:  "",
	}
}

func (r *RR) RemoteAddr() net.Addr {
	return &net.UnixAddr{
		Name: "",
		Net:  "",
	}
}

func (r *RR) SetDeadline(t time.Time) error {
	return nil
}

func (r *RR) SetReadDeadline(t time.Time) error {
	return nil
}

func (r *RR) SetWriteDeadline(t time.Time) error {
	return nil
}

func main() {
	flag.Parse()
	pr, pw := io.Pipe()
	gr, gw := io.Pipe()
	cmd := exec.Command("/abccc/connect")
	cmd.Stdout = pw
	cmd.Stdin = gr
	err2 := cmd.Start()
	if err2 != nil {
		fmt.Println(err2)
	}
	// Set up a connection to the server.
	conn, err := grpc.Dial("", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return &RR{
			pr,
			gw,
		}, nil
	}))
	//conn, err := grpc.Dial("unix:///tmp/abcd.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewFileServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.Watch(ctx, &pb.FileWatchRequest{})
	for {
		recv, err2 := r.Recv()
		if err2 == io.EOF {
			fmt.Println(err2)
		}
		fmt.Println(recv.Path)
	}
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	//log.Printf("Greeting: %s", r.GetMessage())
}
