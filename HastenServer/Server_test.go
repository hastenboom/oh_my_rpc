package HastenServer

import (
	"fmt"
	"net"
	"oh_my_rpc_v2/Common"
	"testing"
)

type Student struct {
	Name string
	Age  int
}

func (s *Student) Foo1(a int, b *string) error {
	*b = fmt.Sprint("hello world: ", a)
	return nil
}

func TestServer(t *testing.T) {

	listen, err := net.Listen("tcp", Common.TEST_ADDR)
	if err != nil {
		return
	}
	server := NewRpcServer()
	server.RegisterService(new(ComputeS1))
	server.Accept(listen)

}
