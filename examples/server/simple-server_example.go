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

func myHandlerForGenericResponses(request interface{}, w http.ResponseWriter, httpRequest *http.Request, responseStruc interface{}) (response interface{}, err error) {
	fooRequest := request.(*Foo1Request)
	fooResponse := &Foo1Response{
		Bar: "Hello \"" + fooRequest.Foo + "\"",
	}
	response = fooResponse
	return
}

// RunServer run a little demo server
func RunServer() {
	soapServer := soap.NewServer()
	soapServer.Log = log.Println
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
	soapServer.RegisterHandlerWithStructTypes("/pathTo", "operationFoo2", "foo2Request", nil, myHandlerForGenericResponses, Foo1Response{}, Foo1Response{})
	err := http.ListenAndServe(":8080", soapServer)
	fmt.Println("exiting with error", err)
}

func main() {
	RunServer()
}
