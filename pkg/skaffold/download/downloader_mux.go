package download

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"io"
)

type DownloaderMux []Downloader

func (dm DownloaderMux) Start(ctx context.Context, builds []graph.Artifact, out io.Writer) error {
	for _, d := range dm {
		if err := d.Start(ctx, builds, out); err != nil {
			return err
		}
	}

	return nil
}

func (dm DownloaderMux) Stop(ctx context.Context) error {
	return nil
}
