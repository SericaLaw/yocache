package yocache

import (
	"bytes"
	"compress/gzip"
)

type Encoder interface {
	Encode(view ByteView) ([]byte, error)
}

type EncoderFunc func(view ByteView) ([]byte, error)
func (f EncoderFunc) Encode(view ByteView) ([]byte, error) {
	return f(view)
}

type Decoder interface {
	Decode(body []byte) (ByteView, error)
}

type DecoderFunc func(body []byte) (ByteView, error)
func (f DecoderFunc) Decode(body []byte) (ByteView, error) {
	return f(body)
}

type Codec interface {
	Encoder
	Decoder
}

type RawCodec struct {}
var _ Codec = (*RawCodec)(nil)

func (c RawCodec) Encode(view ByteView) ([]byte, error) {
	return view.ByteSlice(), nil
}

func (c RawCodec) Decode(body []byte) (ByteView, error) {
	return ByteView{b: body}, nil
}

func RawDecode(body []byte) (ByteView, error) {
	return ByteView{b: body}, nil
}

type GzipCodec struct {}
var _ Codec = (*GzipCodec)(nil)

func (c GzipCodec) Encode(view ByteView) ([]byte, error) {
	var b bytes.Buffer
	writer := gzip.NewWriter(&b)
	defer writer.Close()

	if _, err := writer.Write(view.ByteSlice()); err != nil {
		return nil, err
	}

	if err := writer.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (c GzipCodec) Decode(body []byte) (ByteView, error) {
	b := bytes.NewBuffer(body)

	reader, err := gzip.NewReader(b)
	if err != nil {
		return ByteView{}, err
	}

	var res bytes.Buffer
	if _, err = res.ReadFrom(reader); err != nil {
		return ByteView{}, err
	}

	return ByteView{b: res.Bytes()}, nil
}