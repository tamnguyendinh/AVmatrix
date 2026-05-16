//go:build !ladybugdb

package lbugnative

func OpenWriteRunner(path string) (WriteRunner, error) {
	return nil, ErrUnavailable
}

func OpenWriteRunnerWithEmbeddingDims(path string, embeddingDims int) (WriteRunner, error) {
	return nil, ErrUnavailable
}

func OpenReadRunner(path string) (ReadRunner, error) {
	return nil, ErrUnavailable
}
