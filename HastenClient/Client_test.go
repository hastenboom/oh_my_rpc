package HastenClient

import (
	"log"
	"net"
	"oh_my_rpc_v2/Common"
	"oh_my_rpc_v2/HastenProtocol"
	"oh_my_rpc_v2/HastenServer"
	"testing"
)

func TestClient(t *testing.T) {
	conn, err := net.Dial("tcp", Common.TEST_ADDR)
	if err != nil {
		return
	}

	client, err := NewClient(conn, &HastenProtocol.DefaultOption)
	if err != nil {
		return
	}

	resChan, err := client.Call("ComputeS1.Add", &HastenServer.TwoOperands{
		A: 1,
		B: 2,
	})
	if err != nil {
		return
	}
	res := <-*resChan
	log.Println(res)
}
