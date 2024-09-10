package Client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"oh_my_rpc/Protocol"
	"sync"
)

type RPCall struct {
	Header   *Protocol.Header
	Args     any
	Reply    any
	DoneChan chan *RPCall
}

// used by the client
func (c *RPCall) signalDone() {
	c.DoneChan <- c
}

type Client struct {
	codec          Protocol.Codec
	seq            uint64
	sendingLock    sync.Mutex
	mutex          sync.Mutex
	pendingCallMap map[uint64]*RPCall // seq -> *RPCall
}

var _ io.Closer = (*Client)(nil)

func NewClient(conn net.Conn, option *Protocol.Option) (*Client, error) {
	codec, err := Protocol.CodecFactory(conn, option.CodecType)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(conn).Encode(option)
	log.Println("rpc Client: Send option: ", option)
	if err != nil {
		return nil, err
	}

	client := &Client{
		codec:          codec,
		seq:            1, // 0 means invalid seq
		pendingCallMap: make(map[uint64]*RPCall),
		mutex:          sync.Mutex{},
		sendingLock:    sync.Mutex{},
	}

	return client, nil
}

const (
	connected        = "200 Connected to Gee RPC"
	defaultRPCPath   = "/_geeprc_"
	defaultDebugPath = "/debug/geerpc"
)

func NewHttpClient(conn net.Conn, opt *Protocol.Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", defaultRPCPath))

	// Require successful HTTP response
	// before switching to RPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		return NewClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
}

func (client *Client) SyncCall(serviceMethod string, args any, reply any) {

	header := &Protocol.Header{
		ServiceMethod: serviceMethod,
		Seq:           client.getSeq(),
		Error:         "",
	}

	rpCall := &RPCall{
		Header:   header,
		Args:     args,
		Reply:    reply,
		DoneChan: make(chan *RPCall, 1),
	}

	client.send(rpCall)

	<-rpCall.DoneChan

}

func (client *Client) getSeq() uint64 {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	seq := client.seq
	client.seq += 1
	return seq
}

func (client *Client) Close() error {
	//TODO implement me
	panic("implement me")
}

func (client *Client) send(call *RPCall) {
	client.sendingLock.Lock()
	defer client.sendingLock.Unlock()

	func(*RPCall) {
		client.mutex.Lock()
		defer client.mutex.Unlock()
		client.pendingCallMap[call.Header.Seq] = call
	}(call)

	err := client.codec.Write(call.Header, call.Args)
	if err != nil {
		return
	}

}

// must async call
func (client *Client) handleResponse() {

	for {
		var header *Protocol.Header
		err := client.codec.ReadHeader(header)
		if err != nil {
			return
		}

		rpCall := client.pendingCallMap[header.Seq]
		err = client.codec.ReadBody(rpCall.Reply)
		if err != nil {
			return
		}
	}
}
