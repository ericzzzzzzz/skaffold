package main

import (
	"crypto/md5"
	pb "downstream-sync/filedownload"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	command  []string
	targets  []string
	excludes []string
)

type fileServer struct {
	pb.UnimplementedFileServiceServer
	watcher *fsnotify.Watcher
}

func (s *fileServer) DownloadFile(req *pb.DownloadRequest, stream pb.FileService_DownloadFileServer) error {
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

func executeApp(cmd *cobra.Command, args []string) {
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

	containerCmd := exec.Command(command[0], command[1:]...)
	containerCmd.Stdout = os.Stdout
	containerCmd.Start()
	if err != nil {
		fmt.Println("failed to start app")
		fmt.Println(err)
		return
	}

	grpcServer := grpc.NewServer()
	for _, target := range targets {
		if err := watchDirRecursive(s.watcher, target); err != nil {
			fmt.Printf("failed to watch path: %s, err: %v \n", target, err)
			continue
		}
	}

	pb.RegisterFileServiceServer(grpcServer, &s)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}
func main() {
	var rootCmd = &cobra.Command{
		Use:   "app-server",
		Short: "",
		Long:  `...`,
		Run:   executeApp,
	}

	rootCmd.PersistentFlags().StringSliceVar(&command, "command", []string{}, "List of proxied strings")
	rootCmd.PersistentFlags().StringSliceVar(&targets, "targets", []string{}, "List of target strings")
	rootCmd.PersistentFlags().StringSliceVar(&excludes, "excludes", []string{}, "List of strings to exclude")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func (s *fileServer) Watch(re *pb.FileWatchRequest, stream pb.FileService_WatchServer) error {
	fmt.Println("The following folders are being watched")
	for _, target := range targets {
		fmt.Println("  - " + target)
	}
	fmt.Println("Excludes:")
	for _, ex := range excludes {
		fmt.Println("  - " + ex)
	}

Skip:
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
				if ignore(excludes, event.Name) {
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
				if ignore(excludes, event.Name) {
					continue Skip
				}
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
			if ignore(excludes, path) {
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
	prefixes := []string{"/proc/", "/sys/", "/dev/", "/etc/", "/lib/", "/usr/", "/run/", "/tmp/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) || strings.HasPrefix(path, prefix[1:]) {
			return true
		}
		if path == prefix[:len(prefix)-1] || path == prefix[1:len(prefix)-1] {
			return true
		}
	}

	for _, ex := range excludes {
		ok, err := doublestar.Match(ex, path)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if ok {
			return ok
		}
	}
	return false

}
