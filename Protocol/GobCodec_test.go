package Protocol

import (
	"log"
	"net"
	"testing"
)

func TestGobCodec(t *testing.T) {
	listen, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		return
	}

	conn, err := listen.Accept()
	if err != nil {
		return
	}

	codec := NewGobCodec(conn)

	var header Header
	codec.ReadHeader(&header)
	var arg int
	codec.ReadBody(&arg)

	var header2 Header
	codec.ReadHeader(&header2)
	println()

}

func TestClientCodec(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		return
	}
	codec := NewGobCodec(conn)

	header := Header{
		ServiceMethod: "TestService.TestMethod",
		Error:         "",
		Seq:           1,
	}

	go receive(codec)
	codec.Write(&header, 123)

	codec.Write(&header, 123)

}

func receive(codec Codec) {
	var header Header
	codec.ReadHeader(&header)
	log.Printf("GO: Received header: %v", header)
}
