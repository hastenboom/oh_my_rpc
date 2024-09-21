package HastenRegistry

import (
	"encoding/json"
	"fmt"
	cmap "github.com/orcaman/concurrent-map/v2"
	"net"
	"sync"
	"time"
)

type OpType int

const (
	Registry OpType = iota
	Discovery
	HeartBeat
)

type RegistryReq struct {
	ServiceName string
	OpType      OpType
}

type RegistryResp struct {
	Status string
	Data   any
}

type RegistryServer struct {
	serviceIpMap cmap.ConcurrentMap[string, []string]
	lock         sync.Mutex
}

func StartRegistryServer() *RegistryServer {
	return &RegistryServer{
		serviceIpMap: cmap.New[[]string](),
		lock:         sync.Mutex{},
	}
}

func (r *RegistryServer) Run(registerIpAddr string) {
	listen, err := net.Listen("tcp", registerIpAddr)
	if err != nil {
		return
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			return
		}
		go r.handleConnection(conn)
	}

}

func (r *RegistryServer) handleConnection(conn net.Conn) {

	registryReq := RegistryReq{}
	err := json.NewDecoder(conn).Decode(&registryReq)
	if err != nil {
		return
	}

	serviceName := registryReq.ServiceName
	switch registryReq.OpType {
	case Registry:
		r.handleRegistry(serviceName, conn)
		break

	case Discovery:
		r.handleDiscovery(serviceName, conn)
		conn.Close()
		break
	default:
		json.NewEncoder(conn).Encode("Not a valid operation")
		conn.Close()
	}

}

func (r *RegistryServer) handleRegistry(serviceName string, conn net.Conn) {

	func() {
		r.lock.Lock()
		defer r.lock.Unlock()

		//ipAddr := conn.RemoteAddr().String()
		ipsSplice, ok := r.serviceIpMap.Get(serviceName)
		if !ok {
			ipsSplice = []string{}
		}
		ipsSplice = append(ipsSplice, conn.RemoteAddr().String())
		r.serviceIpMap.Set(serviceName, ipsSplice)

		err := json.NewEncoder(conn).Encode(
			&RegistryResp{
				Status: "200",
				Data:   nil,
			},
		)

		if err != nil {
			return
		}
	}()

	go r.maintainHearBeat(serviceName, conn)

}

func (r *RegistryServer) maintainHearBeat(serviceName string, conn net.Conn) {
	for {

		conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // 设置5秒后的截止时间

		registryReq := RegistryReq{}
		err := json.NewDecoder(conn).Decode(&registryReq)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("读取超时")
				//remove the service
				r.lock.Lock()
				defer r.lock.Unlock()
				ipsSplice, ok := r.serviceIpMap.Get(serviceName)
				if !ok {
					break
				}
				for i, ip := range ipsSplice {
					if ip == conn.RemoteAddr().String() {
						ipsSplice = append(ipsSplice[:i], ipsSplice[i+1:]...)
					}
				}
				break

			} else {
				fmt.Println("解码错误:", err)
				break
			}
		}
	}

}

func (r *RegistryServer) handleDiscovery(serviceName string, conn net.Conn) {

	ipsSlice, ok := r.serviceIpMap.Get(serviceName)
	if !ok {
		err := json.NewEncoder(conn).Encode(
			&RegistryResp{
				Status: "404",
				Data:   nil,
			},
		)
		if err != nil {
			return
		}
	}

	err := json.NewEncoder(conn).Encode(
		&RegistryResp{
			Status: "201",
			Data:   ipsSlice,
		},
	)

	if err != nil {
		return
	}

}
