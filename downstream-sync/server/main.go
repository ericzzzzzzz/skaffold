package main

import (
	"crypto/md5"
	pb "downstream-sync/filedownload"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	listener, err := net.Listen("unix", "/abccc/downstream.sock")
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

	err = watchDirRecursive(s.watcher, dir)
	if err != nil {
		log.Fatalf("Failed to watch: %v", err)
	}
	args := os.Args
	var subProcessArs []string
	if len(args) > 2 {
		subProcessArs = args[2:]
	}

	command := exec.Command(args[1], subProcessArs...)
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

Skip:
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return nil // Watcher closed
			}
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to watch %v", err)
			}
			relPath, err := filepath.Rel(dir, event.Name)
			if err != nil {
				return fmt.Errorf("failed to get relative path, path %s, working dir %s,  %v\n", event.Name, dir, err)
			}

			fileEvent := &pb.FileEvent{
				Path:    relPath,
				Version: 0,
			}

			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				fileEvent.EventType = pb.FileEvent_CREATE
				if ignore(re.Excludes, event.Name) {
					fmt.Println("ignoreeeeee")
					continue Skip
				}

				stat, err := os.Stat(fileEvent.Path)
				if err != nil {
					fmt.Println(err)
					continue Skip
				}
				if stat.IsDir() {
					s.watcher.Add(stat.Name())
				}
			case event.Op&fsnotify.Write == fsnotify.Write:
				fileEvent.EventType = pb.FileEvent_MODIFY
				h, err := HashFile(event.Name)
				if err != nil {
					continue Skip
				}
				fileEvent.MD5Hash = h
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
		if err != nil {
			return err
		}
		if info.IsDir() {
			if ignore(nil, path) {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
}

func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashSum := hash.Sum(nil)

	return fmt.Sprintf("%x", hashSum), nil
}

func ignore(excludes []string, path string) bool {
	prefixes := []string{"/proc/", "/sys/", "/dev/", "/etc/", "/lib/", "/var/", "/usr/", "/run/", "/tmp/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
		if path == prefix[:len(prefix)-1] {
			return true
		}
	}

	dir, err2 := os.Getwd()
	if err2 != nil {
		fmt.Println(err2)
		return false
	}
	rel, err := filepath.Rel(dir, path)

	if err != nil {
		fmt.Println(err)
		return false
	}
	for _, ex := range excludes {
		ok, err := doublestar.Match(ex, rel)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if ok {
			fmt.Println("ignoring due to rule: " + ex)
			return ok
		} else {
			fmt.Println("Not ignoring due to rule: " + ex + "rel: " + rel)

		}
	}
	return false

}
