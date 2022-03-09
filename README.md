# SOAP is dead - long live SOAP

First of all do not write SOAP services if you can avoid it! It is over.

If you can not avoid it this package might help.

## Service

```go
package main

import (
 "encoding/xml"
 "fmt"
 "log"
 "net/http"

 "github.com/3ideas/soap"
)

// FooRequest a simple request
type FooRequest struct {
 XMLName xml.Name `xml:"fooRequest"`
 Foo     string
}

// FooResponse a simple response
type FooResponse struct {
 Bar string
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

 soapServer.RegisterHandler(
  "/",            // Path
  "operationFoo", // SOAPAction
  "fooRequest",   // tagname of soap body content
  func() interface{} { // RequestFactoryFunc - returns struct to unmarshal the request into
   return &FooRequest{}
  },
  // OperationHandlerFunc - do something
  func(request interface{}, w http.ResponseWriter, httpRequest *http.Request) (response interface{}, err error) {
   fooRequest := request.(*FooRequest)
   fooResponse := &FooResponse{
    Bar: "Hello " + fooRequest.Foo,
   }
   response = fooResponse
   return
  },
 )
 err := http.ListenAndServe(":8089", soapServer)
 fmt.Println("exiting with error", err)
}

func main() {
 RunServer()
}

```

## Client

```go
package main

import (
 "context"
 "encoding/xml"
 "log"

 "github.com/3ideas/soap"
)

// FooRequest a simple request
type FooRequest struct {
 XMLName xml.Name `xml:"fooRequest"`
 Foo     string
}

// FooResponse a simple response
type FooResponse struct {
 Bar string
}

func main() {
 client := soap.NewClient("http://127.0.0.1:8080/", nil)
 client.Log = func(msg string, keyString_ValueInterface ...interface{}) {
  keyString_ValueInterface = append(keyString_ValueInterface, msg)
  log.Println(keyString_ValueInterface...)
 } // verbose
 response := &FooResponse{}
 httpResponse, err := client.Call(context.Background(), "operationFoo", &FooRequest{Foo: "hello i am foo"}, response)
 if err != nil {
  panic(err)
 }
 log.Println(response.Bar, httpResponse.Status)
}

```

# Apache License Version 2.0
