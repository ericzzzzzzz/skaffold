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
	"path/filepath"
)

type fileServer struct {
	pb.UnimplementedFileServiceServer
	watcher *fsnotify.Watcher
}

func (s *fileServer) DownloadFile(req *pb.DownloadRequest, stream pb.FileService_DownloadFileServer) error {
	fmt.Println("req.Path::" + req.Path)
	f, err := os.Open(req.Path)
	if err != nil {
		fmt.Println("failed to open")
		return err
	}

	buf := make([]byte, 1024*32)
	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Send the chunk of file
		if err := stream.Send(&pb.DownloadResponse{Chunk: buf[:n]}); err != nil {
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

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get : %v", err)
	}
	fmt.Println("working dir::" + dir)

	err = watchDirRecursive(s.watcher, dir)
	if err != nil {
		log.Fatalf("Failed to watch: %v", err)
	}
	command := exec.Command("bash", "-c", "npm run $NODE_ENV")
	command.Stdout = os.Stdout
	command.Start()
	if err != nil {
		fmt.Println("failed to start app")
		fmt.Println(err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterFileServiceServer(grpcServer, &s)

	fmt.Printf("Server listening at %v\n", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *fileServer) Watch(re *pb.FileWatchRequest, stream pb.FileService_WatchServer) error {
	fmt.Println("received request..")

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
				stat, err := os.Stat(fileEvent.Path)
				if err != nil {
					fmt.Println(err)
					if stat.IsDir() {
						s.watcher.Add(stat.Name())
					}
				}
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

func watchDirRecursive(watcher *fsnotify.Watcher, root string) error {
	err := watcher.Add(root)
	if err != nil {
		return err
	}

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		fmt.Println("added path")
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}
