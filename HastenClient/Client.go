package HastenClient

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"oh_my_rpc_v2/HastenProtocol"
	"oh_my_rpc_v2/HastenRegistry"
	"sync"
)

type Client struct {
	codec       HastenProtocol.RpcCodec
	seq         uint64
	sendingLock sync.Mutex
	mutex       sync.Mutex
	chanMap     map[uint64]*chan *HastenProtocol.RpcProtocol // seq -> *RPCall
}

var _ io.Closer = (*Client)(nil)

func (c *Client) Close() error {
	//TODO implement me
	panic("implement me")
}

func NewClientWithRegistryCenter(
	registryAddr string, serviceName string,
	option *HastenProtocol.Option, balancerType Strategy) (*Client, error) {

	regConn, err := net.Dial("tcp", registryAddr)
	if err != nil {
		return nil, err
	}

	json.NewEncoder(regConn).Encode(HastenRegistry.RegistryReq{
		ServiceName: serviceName,
		OpType:      HastenRegistry.Registry,
	})

	regResp := HastenRegistry.RegistryResp{}
	json.NewDecoder(regConn).Decode(&regResp)

	data := regResp.Data.([]string)
	//balance the ip and create a new client
	balancer := BalancerFactory(balancerType, data)
	serviceIp := balancer.GetNextIp()

	conn, err := net.Dial("tcp", serviceIp)
	if err != nil {
		return nil, err
	}

	return NewClient(conn, option)
}

func NewClient(conn net.Conn, option *HastenProtocol.Option) (*Client, error) {

	codec, err := HastenProtocol.CodecFactory(conn, option.CodecType)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(conn).Encode(option)
	//log.Println("rpc RpcClient: Send option: ", option)

	if err != nil {
		return nil, err
	}

	client := &Client{
		codec:       codec,
		seq:         0,
		chanMap:     make(map[uint64]*chan *HastenProtocol.RpcProtocol),
		mutex:       sync.Mutex{},
		sendingLock: sync.Mutex{},
	}

	go client.handleResponse()

	return client, nil
}

func (c *Client) Call(structMethod string, args any) (*chan *HastenProtocol.RpcProtocol, error) {

	resChan := make(chan *HastenProtocol.RpcProtocol, 1)
	func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.seq++
		c.chanMap[c.seq] = &resChan
	}()

	protocol := &HastenProtocol.RpcProtocol{
		Header: &HastenProtocol.Header{
			StructMethod: structMethod,
			Error:        "",
			Seq:          c.seq,
		},
		Body: args,
	}

	err := c.codec.Write(protocol)
	if err != nil {
		c.chanMap[c.seq] = nil
		close(resChan)
		return nil, err
	}

	return &resChan, nil
}

func (c *Client) getSeq() uint64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	result := c.seq
	c.seq++
	return result
}

func (c *Client) handleResponse() {

	for {
		var h HastenProtocol.Header
		err := c.codec.ReadHeader(&h)
		if err != nil {
			return
		}

		resChan := c.chanMap[h.Seq]
		if resChan == nil {
			var body any
			err = c.codec.ReadBody(&body)
			log.Fatal("rpc RpcClient: Invalid seq: ", h.Seq)
			return
		}

		var res any
		err = c.codec.ReadBody(&res)
		*resChan <- &HastenProtocol.RpcProtocol{
			Header: &h,
			Body:   res,
		}
	}

}
