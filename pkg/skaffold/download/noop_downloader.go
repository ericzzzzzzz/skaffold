package download

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
)

type NoopDownloader struct {
}

func (nd NoopDownloader) Start(ctx context.Context, builds []graph.Artifact) error {
	return nil
}

func (nd NoopDownloader) Stop(ctx context.Context) error {
	return nil
}
