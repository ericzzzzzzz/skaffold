package download

import "context"

type Downloader interface {
	Start(ctx context.Context) error

	Stop(ctx context.Context) error
}
