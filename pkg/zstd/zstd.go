package zstd

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

// Compress input to output.
func Compress(in io.Reader, out io.Writer) error {
	enc, err := zstd.NewWriter(out)
	if err != nil {
		return err
	}
	_, err = io.Copy(enc, in)
	if err != nil {
		enc.Close()
		return err
	}
	return enc.Close()
}

// Create a writer that caches compressors.
// For this operation type we supply a nil Reader.
var encoder, _ = zstd.NewWriter(nil)

// Compress a buffer.
// If you have a destination buffer, the allocation in the call can also be eliminated.
func CompressBytes(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}

func Decompress(in io.Reader, out io.Writer) error {
	d, err := zstd.NewReader(in)
	if err != nil {
		return err
	}
	defer d.Close()

	// Copy content...
	_, err = io.Copy(out, d)
	return err
}

// Create a reader that caches decompressors.
// For this operation type we supply a nil Reader.
var decoder, _ = zstd.NewReader(nil)

// Decompress a buffer. We don't supply a destination buffer,
// so it will be allocated by the decoder.
func DecompressBytes(src []byte) ([]byte, error) {
	return decoder.DecodeAll(src, nil)
}
