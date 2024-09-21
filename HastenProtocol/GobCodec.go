package HastenProtocol

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
	"sync"
)

type GobCodec struct {
	conn      io.ReadWriteCloser
	buf       *bufio.Writer // the buf is derived from the conn
	decoder   *gob.Decoder
	encoder   *gob.Encoder
	writeLock sync.Locker
}

var _ RpcCodec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) RpcCodec {
	return &GobCodec{
		conn:      conn,
		buf:       bufio.NewWriter(conn),
		decoder:   gob.NewDecoder(conn),
		encoder:   gob.NewEncoder(conn),
		writeLock: &sync.Mutex{},
	}
}

func (g *GobCodec) ReadServiceName(serviceName *string) error {
	err := g.decoder.Decode(serviceName)
	if err != nil {
		return err
	}
	return nil
}

func (g *GobCodec) WriteServiceName(serviceName string) error {
	g.writeLock.Lock()
	defer g.writeLock.Unlock()

	err := g.encoder.Encode(serviceName)
	if err != nil {
		return err
	}
	return nil
}

func (g *GobCodec) Close() error {
	err := g.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

func (g *GobCodec) ReadHeader(header *Header) error {
	err := g.decoder.Decode(header)
	log.Println("rpc: gob read header:", header)
	return err
}

/*
ReadBody
in Server side: the body is called reply
in client side: the body is called args
*/
func (g *GobCodec) ReadBody(body any) error {
	err := g.decoder.Decode(body)
	log.Println("rpc: gob read body:", body)
	return err
}

func (g *GobCodec) Write(rpcProtocol *RpcProtocol) error {
	g.writeLock.Lock()
	defer g.writeLock.Unlock()

	h := rpcProtocol.Header
	body := rpcProtocol.Body

	var err error

	defer func() {
		_ = g.buf.Flush()
		if err != nil {
			_ = g.Close()
		}
	}()

	if err = g.encoder.Encode(h); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return err
	}
	log.Println("rpc: gob write header:", h)

	if err = g.encoder.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return err
	}
	log.Println("rpc: gob write body:", body)

	return err
}

/*func (g *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = g.buf.Flush()
		if err != nil {
			_ = g.Close()
		}
	}()
	if err = g.encoder.Encode(h); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return
	}
	log.Println("rpc: gob write header:", h)
	if err = g.encoder.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return
	}
	log.Println("rpc: gob write body:", body)
	return
}
*/
