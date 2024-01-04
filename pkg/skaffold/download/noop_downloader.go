package download

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"io"
)

type NoopDownloader struct {
}

func (nd NoopDownloader) Start(ctx context.Context, builds []graph.Artifact, out io.Writer) error {
	return nil
}

func (nd NoopDownloader) Stop(ctx context.Context) error {
	return nil
}
