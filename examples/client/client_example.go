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
		var values []interface{}
		values = append(values, "SOAP Client: ")
		values = append(values, keyString_ValueInterface...)
		log.Println(values...)
	}
	client.LogLevel = 2

	response := &FooResponse{}
	httpResponse, err := client.Call(context.Background(), "operationFoo", &FooRequest{Foo: "hello i am foo"}, response)
	if err != nil {
		panic(err)
	}
	log.Println(response.Bar, httpResponse.Status)
}
