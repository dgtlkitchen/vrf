package codec

import (
	"fmt"

	cometabci "github.com/cometbft/cometbft/abci/types"
)

// Compressor is a small interface for wrapping encoding with compression.
type Compressor interface {
	Compress(bz []byte) ([]byte, error)
	Decompress(bz []byte) ([]byte, error)
}

// CompressionExtendedCommitCodec wraps an ExtendedCommitCodec with a Compressor.
type CompressionExtendedCommitCodec struct {
	base       ExtendedCommitCodec
	compressor Compressor
}

func NewCompressionExtendedCommitCodec(base ExtendedCommitCodec, compressor Compressor) *CompressionExtendedCommitCodec {
	return &CompressionExtendedCommitCodec{
		base:       base,
		compressor: compressor,
	}
}

func (codec *CompressionExtendedCommitCodec) Encode(ec cometabci.ExtendedCommitInfo) ([]byte, error) {
	raw, err := codec.base.Encode(ec)
	if err != nil {
		return nil, err
	}

	bz, err := codec.compressor.Compress(raw)
	if err != nil {
		return nil, fmt.Errorf("compress extended commit info: %w", err)
	}

	return bz, nil
}

func (codec *CompressionExtendedCommitCodec) Decode(bz []byte) (cometabci.ExtendedCommitInfo, error) {
	raw, err := codec.compressor.Decompress(bz)
	if err != nil {
		return cometabci.ExtendedCommitInfo{}, fmt.Errorf("decompress extended commit info: %w", err)
	}

	ec, err := codec.base.Decode(raw)
	if err != nil {
		return cometabci.ExtendedCommitInfo{}, err
	}

	return ec, nil
}
