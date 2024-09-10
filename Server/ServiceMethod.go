package Server

import (
	"go/ast"
	"log"
	"reflect"
)

type RpcFunc func(in interface{}, out interface{}) error

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

/*----------------*/

type service struct {
	serviceName  string                     //a.k.a struct name
	serviceType  reflect.Type               //a.k.a struct type
	serviceValue reflect.Value              //a.k.a struct value
	methodMap    map[string]*reflect.Method //a.k.a method map
}

func newService[T any](serviceValue T) *service {
	sPtr := new(service)
	sPtr.serviceValue = reflect.ValueOf(serviceValue)
	//
	sPtr.serviceName = reflect.Indirect(sPtr.serviceValue).Type().Name()
	sPtr.serviceType = reflect.TypeOf(serviceValue)
	if !ast.IsExported(sPtr.serviceName) {
		log.Fatalf("rpc Server: %sPtr is not a valid service name", sPtr.serviceName)
	}
	sPtr.registerMethods()
	return sPtr
}

func (s *service) registerMethods() {
	s.methodMap = make(map[string]*reflect.Method)

	for i := 0; i < s.serviceType.NumMethod(); i++ {
		method := s.serviceType.Method(i)
		mType := method.Type
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}

		s.methodMap[method.Name] = &method
		log.Printf("rpc Server: register %s.%s\n", s.serviceName, method.Name)
	}
}

func (s *service) call(m *reflect.Method, argv reflect.Value, replyv reflect.Value) error {

	returnError := m.Func.Call([]reflect.Value{s.serviceValue, argv, replyv})
	if err := returnError[0].Interface(); err != nil {
		return err.(error)
	}
	return nil
}

/*--------------------------*/

func newArgv(argType reflect.Type) reflect.Value {
	var argv reflect.Value

	if argType.Kind() == reflect.Ptr {
		argv = reflect.New(argType.Elem()) //the pointer
	} else {
		argv = reflect.New(argType).Elem()
	}
	return argv
}

func newReplyv(replyType reflect.Type) reflect.Value {

	//reply must be a pointer here
	/*	if replyType.Elem().Kind() != reflect.Ptr {
		panic("rpc server: reply type must be a pointer")
	}*/

	replyValue := reflect.New(replyType.Elem())

	switch replyType.Elem().Kind() {
	case reflect.Map:
		replyValue.Elem().Set(reflect.MakeMap(replyType.Elem()))
	case reflect.Slice:
		replyValue.Elem().Set(reflect.MakeSlice(replyType.Elem(), 0, 0))
	}
	return replyValue
}
