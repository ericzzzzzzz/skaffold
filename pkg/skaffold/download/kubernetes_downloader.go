package download

import (
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/proto/filedownload"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type KubernetesDownloader struct {
	artifacts []*latest.Artifact
	*kubectl.CLI
}

func (kd KubernetesDownloader) Start(ctx context.Context, builds []graph.Artifact, out io.Writer) error {

	go func() {
		kClient, err2 := kubernetesclient.Client(kd.KubeContext)
		if err2 != nil {
			fmt.Println(err2)
		}
		list, _ := kClient.CoreV1().Pods(kd.Namespace).List(ctx, metav1.ListOptions{})

		for _, p := range list.Items {
			for _, c := range p.Spec.Containers {
				if ds := getDownstreamSync(ctx, kd.artifacts, builds, c.Image); ds != nil {
					pr, pw := io.Pipe()
					gr, gw := io.Pipe()
					command := kd.CLI.Command(ctx, "exec", "-it", "pods/"+p.Name, "-c", c.Name, "--", "/abccc/app-connect")
					command.Stdout = pw
					command.Stdin = gr
					err := command.Start()
					if err != nil {
						fmt.Println("failed to connect the remote ")
						fmt.Println(err)
					}
					conn, err := grpc.DialContext(ctx, "", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
						return &Conn{pr, gw}, nil
					}))
					if err != nil {
						fmt.Println(err)
					}

					client := filedownload.NewFileServiceClient(conn)

					watch, err := client.Watch(ctx, &filedownload.FileWatchRequest{Excludes: ds.Excludes})
					if err != nil {
						fmt.Println(err)
					}

					go func() {
						for {
							recv, err2 := watch.Recv()
							if err2 != nil {
								fmt.Println(err2)
								return
							}

							for _, entry := range ds.Entry {

								if !MatchDir(entry.RemoteSrc, recv.Path) {
									continue
								}
								rel, err2 := filepath.Rel(entry.RemoteSrc, recv.Path)
								if err2 != nil {
									fmt.Println(err2)
									continue
								}

								t := filepath.Join(entry.LocalDst, rel)
								if v, ok := filemon.SyncedHash[t]; ok {
									if v == recv.MD5Hash {
										output.Default.Fprintf(out, "File %s already synced. \n", t)
										continue
									}
								}
								filemon.SyncedHash[t] = recv.MD5Hash
								output.Default.Fprintf(out, "Downloading %s from remote path %s \n", t, recv.Path)

								file, err2 := client.DownloadFile(context.Background(), &filedownload.DownloadRequest{Path: recv.Path})
								if err2 != nil {
									fmt.Println(err2)
								}

								os.MkdirAll(filepath.Dir(t), 0755)
								create, err2 := os.Create(t)
								if err2 != nil {
									fmt.Println("failed to create")
									fmt.Println(err2)
								}
								for {
									response, err2 := file.Recv()
									if err2 == io.EOF {
										break
									}
									create.Write(response.Chunk)
								}
								create.Close()

							}

						}
					}()
				}
			}
		}
	}()

	return nil
}

func (kd KubernetesDownloader) Stop(ctx context.Context) error {
	return nil
}

func getDownstreamSync(ctx context.Context, artifacts []*latest.Artifact, builds []graph.Artifact, containerImage string) *latest.DownstreamSync {

	g := graph.ToArtifactGraph(artifacts)

	for _, b := range builds {
		if b.Tag != containerImage {
			continue
		}
		if v, ok := g[b.ImageName]; ok {
			return v.DownstreamSync
		}
	}
	return nil
}

func MatchDir(targetDir string, changedDir string) bool {
	if targetDir == "." {
		return true
	}
	list1 := strings.Split(filepath.Clean(targetDir), string(os.PathSeparator))
	list2 := strings.Split(filepath.Clean(changedDir), string(os.PathSeparator))

	if len(list1) > len(list2) {
		return false
	}
	for i, ele := range list1 {
		if list2[i] != ele {
			return false
		}
	}
	return true
}

type Conn struct {
	*io.PipeReader
	*io.PipeWriter
}

func (c *Conn) LocalAddr() net.Addr {
	return &net.UnixAddr{
		Name: "",
		Net:  "Unix",
	}

}

func (c *Conn) RemoteAddr() net.Addr {
	return &net.UnixAddr{
		Name: "",
		Net:  "Unix",
	}
}

func (c *Conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *Conn) Close() error {
	err := c.PipeReader.Close()
	if err != nil {
		return err
	}
	return c.PipeWriter.Close()
}
