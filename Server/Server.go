package Server

import (
	"encoding/json"
	"errors"
	cmap "github.com/orcaman/concurrent-map/v2"
	"io"
	"log"
	"net"
	"oh_my_rpc/Protocol"
	"reflect"
	"strings"
	"sync"
)

type RpcServer struct {
	serviceMap cmap.ConcurrentMap[string, *service]
}

func NewRpcServer() *RpcServer {
	return &RpcServer{
		serviceMap: cmap.New[*service](),
	}
}

func (server *RpcServer) Accept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:" + conn.RemoteAddr().String())
		}

		//go server.handleConnection(conn)
		//TODO: for debug
		server.handleConnection(conn)
	}
}

func (server *RpcServer) handleConnection(conn net.Conn) {
	/*pre check*/
	opt := new(Protocol.Option)
	err := server.validateOption(conn, opt)
	if err != nil {
		return
	}

	codec, err := Protocol.CodecFactory(conn, opt.CodecType)
	if err != nil {
		return
	}

	server.handleRpcRequest(codec)

}

// aim to every connection
func (server *RpcServer) validateOption(conn net.Conn, opt *Protocol.Option) error {
	err := json.NewDecoder(conn).Decode(opt)
	log.Println("rpc Server: Received option: ", opt)
	if err != nil {
		log.Println("rpc Server: Error decoding option:" + err.Error())
		return err
	}

	if opt.MagicNumber != Protocol.DefaultMagicNumber {
		log.Println("rpc Server: Invalid magic number")
		return err
	}
	return nil
}

type request struct {
	header  *Protocol.Header
	argv    reflect.Value
	replyv  reflect.Value
	method  *reflect.Method
	service *service
}

var invalidReqBody = struct{}{}

func (server *RpcServer) handleRpcRequest(codec Protocol.Codec) {
	// a lock for sending response in one specific connection
	sendingLock := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	defer func(codec Protocol.Codec) {
		err := codec.Close()
		if err != nil {
			//TODO: close failed ... what should we do?
		}
	}(codec)

	for {
		req, err := server.getRequest(codec)
		if err != nil { //header decode went wrong
			if req == nil { //no req, faulted data received? Not, actually, it's something like EOF
				break
			}
			req.header.Error = err.Error()
			server.sendRpcResponse(codec, req.header, invalidReqBody, sendingLock)
			continue
		}
		wg.Add(1)
		go server.doHandleRpcRequest(codec, req, sendingLock, wg)
	}

	wg.Wait()
}

func (server *RpcServer) getRequest(codec Protocol.Codec) (*request, error) {
	var header Protocol.Header
	err := codec.ReadHeader(&header)
	if err != nil {
		if err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
			log.Println("rpc server: get header error:", err)
		}
	}

	/*handle body. The client body is the argv*/
	service, method, err := server.findService(header.ServiceMethod)
	if err != nil {
		return nil, err
	}

	//parts of the protocol
	argv := newArgv(method.Type.In(1))
	replyv := newReplyv(method.Type.In(2))

	// the ReadBody receive only the pointer of the argv
	argvAny := argv.Interface()
	if argv.Type().Kind() != reflect.Ptr {
		argvAny = argv.Addr().Interface()
	}

	//the argv now is injected
	err = codec.ReadBody(argvAny)
	if err != nil {
		log.Println("rpc server: get body error:", err)
	}

	return &request{
		header:  &header,
		argv:    argv,
		replyv:  replyv,
		method:  method,
		service: service,
	}, nil
}

func (server *RpcServer) findService(serviceMethod string) (*service, *reflect.Method, error) {
	dotIndex := strings.LastIndex(serviceMethod, ".")
	if dotIndex < 0 {
		return nil, nil, errors.New("rpc server: invalid anyObj method format")
	}

	serviceName, methodName := serviceMethod[:dotIndex], serviceMethod[dotIndex+1:]

	service, isExist := server.serviceMap.Get(serviceName)
	if !isExist {
		return nil, nil, errors.New("rpc server:  service:" + serviceName + " not found")
	}

	method := service.methodMap[methodName]
	if method == nil {
		return nil, nil, errors.New("rpc server: method:" + methodName + " not found in service:" + serviceName)
	}
	return service, method, nil

}
func (server *RpcServer) RegisterService(structObj any) error {
	serviceObjPtr := newService(structObj)

	ok := server.serviceMap.SetIfAbsent(serviceObjPtr.serviceName, serviceObjPtr)
	if !ok {
		return errors.New("rpc server: serviceObjPtr already registered")
	}
	return nil
}

func (server *RpcServer) sendRpcResponse(
	codec Protocol.Codec, header *Protocol.Header,
	reply any, sendingLock *sync.Mutex) {

	sendingLock.Lock()
	defer sendingLock.Unlock()

	err := codec.Write(header, reply)
	if err != nil {
		log.Println("rpc server: write response error:", err)
		return
	}
}

func (server *RpcServer) doHandleRpcRequest(codec Protocol.Codec, req *request, sendingLock *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	err := req.service.call(req.method, req.argv, req.replyv)

	if err != nil {
		req.header.Error = err.Error()
		server.sendRpcResponse(codec, req.header, invalidReqBody, sendingLock)
		return
	}

	//server send the replyv as the body
	server.sendRpcResponse(codec, req.header, req.replyv.Interface(), sendingLock)

}

//type request struct {
//	header  *Protocol.Header
//	argv    reflect.Value
//	replyv  reflect.Value
//	method  *reflect.Method
//	service *service
//}
//
//var invalidReqBody = struct{}{}
//
//func (server *RpcServer) handleRpcRequest(codec Protocol.Codec) {
//	// a lock for sending response in one specific connection
//	sendingLock := new(sync.Mutex)
//	wg := new(sync.WaitGroup)
//
//	defer func(codec Protocol.Codec) {
//		err := codec.Close()
//		if err != nil {
//			//TODO: close failed ... what should we do?
//		}
//	}(codec)
//
//	for {
//		req, err := server.getRequest(codec)
//		/*FIXME: this logic is buggy, fix that combining with the getRequest()*/
//		if err != nil { //header decode went wrong
//			if req == nil { //no req, faulted data received? Not, actually, it's something like EOF
//				break
//			}
//			req.header.Error = err.Error()
//			server.sendRpcResponse(codec, req.header, invalidReqBody, sendingLock)
//			continue
//		}
//
//		wg.Add(1)
//		//the real part that implements the RPC call
//		go server.doHandleRpcRequest(codec, req, sendingLock, wg)
//	}
//	wg.Wait()
//
//}
//
//func (server *RpcServer) getRequest(codec Protocol.Codec) (*request, error) {
//
//	var header *Protocol.Header
//
//	/*handle header*/
//	err := codec.ReadHeader(header)
//	if err != nil {
//		if err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
//			log.Println("rpc server: get header error:", err)
//		}
//		//TODO: also end the connection here, but send some error mes back to client first
//		return nil, err
//	}
//
//	/*handle body. The client body is the argv*/
//	service, method, err := server.findService(header.ServiceMethod)
//	if err != nil {
//		return nil, err
//	}
//
//	//parts of the protocol
//	argv := newArgv(method.Type.In(1))
//	replyv := newReplyv(method.Type.In(2))
//
//	// the ReadBody receive only the pointer of the argv
//	argvAny := argv.Interface()
//	if argv.Type().Kind() != reflect.Ptr {
//		argvAny = argv.Addr().Interface()
//	}
//
//	//the argv now is injected
//	err = codec.ReadBody(argvAny)
//	if err != nil {
//		log.Println("rpc server: get body error:", err)
//	}
//
//	return &request{
//		header:  header,
//		argv:    argv,
//		replyv:  replyv,
//		method:  method,
//		service: service,
//	}, nil
//
//}
//
//func (server *RpcServer) findService(serviceMethod string) (*service, *reflect.Method, error) {
//	dotIndex := strings.LastIndex(serviceMethod, ".")
//	if dotIndex < 0 {
//		return nil, nil, errors.New("rpc server: invalid anyObj method format")
//	}
//
//	serviceName, methodName := serviceMethod[:dotIndex], serviceMethod[dotIndex+1:]
//
//	service, isExist := server.serviceMap.Get(serviceName)
//	if !isExist {
//		return nil, nil, errors.New("rpc server:  service:" + serviceName + " not found")
//	}
//
//	method := service.methodMap[methodName]
//	if method == nil {
//		return nil, nil, errors.New("rpc server: method:" + methodName + " not found in service:" + serviceName)
//	}
//	return service, method, nil
//
//}
//
//func (server *RpcServer) doHandleRpcRequest(codec Protocol.Codec, req *request, sendingLock *sync.Mutex, wg *sync.WaitGroup) {
//	defer wg.Done()
//
//	err := req.service.call(req.method, req.argv, req.replyv)
//	if err != nil {
//		req.header.Error = err.Error()
//		server.sendRpcResponse(codec, req.header, invalidReqBody, sendingLock)
//		return
//	}
//
//	//server send the replyv as the body
//	server.sendRpcResponse(codec, req.header, req.replyv.Interface(), sendingLock)
//}
//
//func (server *RpcServer) sendRpcResponse(
//	codec Protocol.Codec, header *Protocol.Header,
//	reply any, sendingLock *sync.Mutex) {
//	sendingLock.Lock()
//	defer sendingLock.Unlock()
//
//	err := codec.Write(header, reply)
//	if err != nil {
//		log.Println("rpc server: write response error:", err)
//		return
//	}
//}
//
