package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"

	"github.com/3ideas/soap"
)

// FooRequest a simple request
type Foo1Request struct {
	XMLName xml.Name `xml:"fooRequest"`
	Foo     string
}

// FooResponse a simple response
type Foo1Response struct {
	Bar string
}

// Any type we want to pass in to the resonse handler
type FooResponseInfo struct {
	Info string
}

func myHandlerForResponses(request interface{}, w http.ResponseWriter, httpRequest *http.Request, info interface{}) (response interface{}, err error) {
	myInfo := info.(*FooResponseInfo)
	fooRequest := request.(*Foo1Request)
	fooResponse := &Foo1Response{
		Bar: "Hello \"" + myInfo.Info + fooRequest.Foo + "\"",
	}
	response = fooResponse
	return
}

// RunServer run a little demo server
func RunServer() {
	soapServer := soap.NewServer()
	soapServer.Log = func(msg string, keyString_ValueInterface ...interface{}) {
		keyString_ValueInterface = append(keyString_ValueInterface, msg)
		var values []interface{}
		values = append(values, "SOAP Server: ")
		values = append(values, keyString_ValueInterface...)
		log.Println(values...)
	}
	soapServer.LogLevel = 2
	soapServer.RegisterHandler(
		"/pathTo",
		"operationFoo", // SOAPAction
		"fooRequest",   // tagname of soap body content
		// RequestFactoryFunc - give the server sth. to unmarshal the request into
		func() interface{} {
			return &Foo1Request{}
		},
		// OperationHandlerFunc - do something
		func(request interface{}, w http.ResponseWriter, httpRequest *http.Request) (response interface{}, err error) {
			fooRequest := request.(*Foo1Request)
			fooResponse := &Foo1Response{
				Bar: "Hello \"" + fooRequest.Foo + "\"",
			}
			response = fooResponse
			return
		},
	)
	soapServer.RegisterHandlerWithInfo("/pathTo", "operationFoo2", "foo2Request", nil, myHandlerForResponses, Foo1Response{}, &FooResponseInfo{Info: "some info"})
	err := http.ListenAndServe(":8080", soapServer)
	fmt.Println("exiting with error", err)
}

func main() {
	RunServer()
}
