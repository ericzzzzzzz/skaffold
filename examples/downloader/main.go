package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func main() {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	temp, err := os.Create("/tmp/sync-log.txt")
	//temp, err := os.CreateTemp("/tmp", "")
	// Add the directory to watch
	err = watcher.Add("anything")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(temp.Name())

	fmt.Println("Watching for file changes...")
	for _, x := range watcher.WatchList() {
		fmt.Println(x)
	}

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				fmt.Println("Event:", event)
				x := &FileChange{
					Path:              event.Name,
					EventReceivedTime: time.Now(),
					Hash:              0,
					OP:                "Write",
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("Modified file:", event.Name)
					// temp file, ignore
					if strings.HasSuffix(event.Name, "~") {
						continue
					}
					x.OP = "Write"
					marshal, _ := json.Marshal(x)
					_, err2 := temp.Write(marshal)
					temp.WriteString("\n")

					if err2 != nil {
						fmt.Println("failed to write..")
					}
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					x.OP = "Create"
					marshal, _ := json.Marshal(x)
					_, err2 := temp.Write(marshal)
					temp.WriteString("\n")
					if err2 != nil {
						fmt.Println("failed to write..")
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("Error:", err)
			}
		}
	}()

	<-done
}

type FileChange struct {
	Path              string    `json:"path"`
	EventReceivedTime time.Time `json:"eventReceivedTime"`
	Hash              int64     `json:"hash"`
	OP                string    `json:"op"`
}

func hashFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}
