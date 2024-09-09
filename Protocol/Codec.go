package Protocol

import (
	"io"
	"net"
)

type Header struct {
	ServiceMethod string
	Error         string
	Seq           uint64 // identify each request
}

type Codec interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(body any) error
	Write(*Header, any) error
}

type SerializerEnum string

const (
	GobType  SerializerEnum = "application/gob"
	JsonType SerializerEnum = "application/json"
)

func CodecFactory(conn net.Conn, serializerType SerializerEnum) (Codec, error) {
	switch serializerType {
	case GobType:
		return NewGobCodec(conn), nil
	case JsonType:
		panic("JsonType not implemented yet")
	default:
		panic("Unknown serializer type")
	}
}
