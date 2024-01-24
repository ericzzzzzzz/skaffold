package download

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type Downloader interface {
	Start(ctx context.Context, builds []graph.Artifact) error

	Stop(ctx context.Context) error
}

func NewKubernetesDownloader(artifacts []*latest.Artifact, cli *kubectl.CLI) Downloader {
	return KubernetesDownloader{
		artifacts: artifacts,
		CLI:       cli,
	}
}

func NewNoopDownloader() Downloader {
	return NoopDownloader{}
}
