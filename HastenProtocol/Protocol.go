package HastenProtocol

import (
	"io"
	"net"
)

type Header struct {
	StructMethod string
	Error        string
	Seq          uint64 // identify each request
}

type RpcProtocol struct {
	Header *Header
	Body   any
}

type RegistryProtocol struct {
	Service string
}

type RpcCodec interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(body any) error
	Write(*RpcProtocol) error
	ReadServiceName(serviceName *string) error
	WriteServiceName(serviceName string) error
}

type CodecEnum string

const (
	GobType  CodecEnum = "application/gob"
	JsonType CodecEnum = "application/json"
)

func CodecFactory(conn net.Conn, serializerType CodecEnum) (RpcCodec, error) {
	switch serializerType {
	case GobType:
		return NewGobCodec(conn), nil
	case JsonType:
		panic("JsonType not implemented yet")
	default:
		panic("Unknown serializer type")
	}
}

/*------------*/

type Option struct {
	MagicNumber int
	CodecType   CodecEnum // supporting only the gob for now
}

const DefaultMagicNumber = 0x3bef5c

var DefaultOption = Option{
	MagicNumber: DefaultMagicNumber,
	CodecType:   GobType,
}
