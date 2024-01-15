package main

import (
	pb "downstream-sync/filedownload"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
)

type fileServer struct {
	pb.UnimplementedFileServiceServer
	watcher *fsnotify.Watcher
}

func (s *fileServer) DownloadFile(req *pb.DownloadRequest, stream pb.FileService_DownloadFileServer) error {
	f, err := os.Open(req.Path)

	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	for {
		reader := io.NewSectionReader(f, 0, 1024)
		_, err := reader.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if err := stream.Send(&pb.DownloadResponse{Chunk: buf}); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// command
	listener, err := net.Listen("unix", "/tmp/downstream.sock") // You can change the port if needed
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()
	s := fileServer{
		watcher: watcher,
	}
	err = s.watcher.Add(".")
	if err != nil {
		log.Fatalf("Failed to watch: %v", err)
	}
	err = exec.Command("./app").Start()
	if err != nil {
		fmt.Println("failed to start app")
		fmt.Println(err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterFileServiceServer(grpcServer, &s)

	log.Printf("Server listening at %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *fileServer) Watch(re *pb.FileWatchRequest, stream pb.FileService_WatchServer) error {

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return nil // Watcher closed
			}

			fileEvent := &pb.FileEvent{
				Path:    event.Name,
				Version: 0,
			}

			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				fileEvent.EventType = pb.FileEvent_CREATE
			case event.Op&fsnotify.Write == fsnotify.Write:
				fileEvent.EventType = pb.FileEvent_MODIFY
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				fileEvent.EventType = pb.FileEvent_DELETE
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				fileEvent.EventType = pb.FileEvent_RENAME
			}
			if fileEvent.EventType != pb.FileEvent_MODIFY {
				fmt.Printf("Non-Modify event %v \n", fileEvent)
				continue
			}

			if err := stream.Send(fileEvent); err != nil {
				return fmt.Errorf("failed to send event: %v", err)
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return nil // Watcher closed
			}
			log.Println("error:", err)
		}
	}
}
