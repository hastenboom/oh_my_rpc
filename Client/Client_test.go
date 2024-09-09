package Client

import (
	"net"
	"oh_my_rpc/Common"
	"oh_my_rpc/Protocol"
	"testing"
)

func TestClient(t *testing.T) {
	conn, err := net.Dial("tcp", Common.Test_ADDR)
	if err != nil {
		return
	}

	client, err := NewClient(conn, &Protocol.DefaultOption)
	if err != nil {
		return
	}
	var result string
	client.SyncCall("Student.Foo1", 123, &result)

}
