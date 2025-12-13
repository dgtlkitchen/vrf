package codec

import "github.com/DataDog/zstd"

type ZStdCompressor struct{}

func NewZStdCompressor() *ZStdCompressor { return &ZStdCompressor{} }

func (c *ZStdCompressor) Compress(bz []byte) ([]byte, error) {
	if len(bz) == 0 {
		return nil, nil
	}

	return zstd.Compress(nil, bz)
}

func (c *ZStdCompressor) Decompress(bz []byte) ([]byte, error) {
	if len(bz) == 0 {
		return nil, nil
	}

	return zstd.Decompress(nil, bz)
}
