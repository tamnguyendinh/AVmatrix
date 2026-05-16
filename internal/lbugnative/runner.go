package lbugnative

import (
	"errors"

	"github.com/tamnguyendinh/avmatrix-go/internal/lbugruntime"
)

var ErrUnavailable = errors.New("native LadybugDB runner is unavailable; build with -tags ladybugdb")

type WriteRunner interface {
	Query(query string) error
	QueryRows(query string) ([]lbugruntime.Row, error)
	Close() error
}

type ReadRunner interface {
	QueryRows(query string) ([]lbugruntime.Row, error)
	Close() error
}
