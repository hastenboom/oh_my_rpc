package Registry

import (
	cmap "github.com/orcaman/concurrent-map/v2"
	"log"
	"net"
)

type RegistryServer struct {
	serverIpMap cmap.ConcurrentMap[string, []string]
}

func NewRegistryServer() *RegistryServer {

	return &RegistryServer{
		serverIpMap: cmap.New[[]string](),
	}

}

func (r *RegistryServer) Accept(listener net.Listener) {

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:" + conn.RemoteAddr().String())
		}

		//go server.handleConnection(conn)
		//TODO: for debug
	}

}
