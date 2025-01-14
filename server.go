package soap

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
)

// OperationHandlerFunc runs the actual business logic - request is whatever you constructed in RequestFactoryFunc
type OperationHandlerFunc func(request interface{}, w http.ResponseWriter, httpRequest *http.Request) (response interface{}, err error)

// OperationHandlerFuncWithInfo A modified operational handler function that also passes in an optional information field that can be used to pass in extra information about the request.
// Its type is upto the handler to define.
type OperationHandlerFuncWithInfo func(request interface{}, w http.ResponseWriter, httpRequest *http.Request, handlerInfo interface{}) (response interface{}, err error)

// RequestFactoryFunc constructs a request object for OperationHandlerFunc
type RequestFactoryFunc func() interface{}

type dummyContent struct{}

type operationHandler struct {
	requestFactory  RequestFactoryFunc
	handler         OperationHandlerFunc
	handlerWithInfo OperationHandlerFuncWithInfo
	requestStruct   interface{}
	handlerInfo     interface{}
}

type responseWriter struct {
	logFunc       func(msg string, keyString_ValueInterface ...interface{})
	logLevel      int
	w             http.ResponseWriter
	outputStarted bool
}

func (w *responseWriter) log(level int, msg string, args ...interface{}) {
	if w.logFunc != nil && level <= w.logLevel {
		w.logFunc(msg, args...)
	}
}

func (w *responseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.outputStarted = true

	w.log(2, "Response writter", "writing response: ", string(b))

	return w.w.Write(b)
}

func (w *responseWriter) WriteHeader(code int) {
	w.w.WriteHeader(code)
}

// Server a SOAP server, which can be run standalone or used as a http.HandlerFunc
type Server struct {
	//Log         func(...interface{}) // do nothing on nil or add your fmt.Print* or log.*
	Log         func(msg string, keyString_ValueInterface ...interface{})
	handlers    map[string]map[string]map[string]*operationHandler
	Marshaller  XMLMarshaller
	ContentType string
	SoapVersion string
	LogLevel    int
}

// NewServer construct a new SOAP server
func NewServer() *Server {
	return &Server{
		handlers:    make(map[string]map[string]map[string]*operationHandler),
		Marshaller:  defaultMarshaller{},
		ContentType: SoapContentType11,
		SoapVersion: SoapVersion11,
		LogLevel:    0,
	}
}

func (s *Server) log(level int, msg string, args ...interface{}) {
	if s.Log != nil && level <= s.LogLevel {
		s.Log(msg, args...)
	}
}

func (s *Server) UseSoap11() {
	s.SoapVersion = SoapVersion11
	s.ContentType = SoapContentType11
}

func (s *Server) UseSoap12() {
	s.SoapVersion = SoapVersion12
	s.ContentType = SoapContentType12
}

// RegisterHandler register to handle an operation. This function must not be
// called after the server has been started.
func (s *Server) RegisterHandler(path string, action string, messageType string, requestFactory RequestFactoryFunc, operationHandlerFunc OperationHandlerFunc) {
	if _, ok := s.handlers[path]; !ok {
		s.handlers[path] = make(map[string]map[string]*operationHandler)
	}

	if _, ok := s.handlers[path][action]; !ok {
		s.handlers[path][action] = make(map[string]*operationHandler)
	}
	s.handlers[path][action][messageType] = &operationHandler{
		handler:        operationHandlerFunc,
		requestFactory: requestFactory,
	}
}

// RegisterHandlerWithInfo Register a slightly differenct handler function, to enable auto generation of the Reqest struct from the type.
// also the modiled
// This function must not be called after the server has been started.
func (s *Server) RegisterHandlerWithInfo(path string, action string, messageType string, requestFactory RequestFactoryFunc, operationHandlerFunc OperationHandlerFuncWithInfo, reqStruct interface{}, handlerInfo interface{}) {
	if _, ok := s.handlers[path]; !ok {
		s.handlers[path] = make(map[string]map[string]*operationHandler)
	}

	if _, ok := s.handlers[path][action]; !ok {
		s.handlers[path][action] = make(map[string]*operationHandler)
	}
	s.handlers[path][action][messageType] = &operationHandler{
		handlerWithInfo: operationHandlerFunc,
		requestFactory:  requestFactory,
		requestStruct:   reqStruct,
		handlerInfo:     handlerInfo,
	}
}

func (s *Server) handleError(err error, w http.ResponseWriter) {
	// has to write a soap fault
	s.log(1, "error", "handling error:", err)
	responseEnvelope := &Envelope{
		Body: Body{
			Content: &Fault{
				String: err.Error(),
			},
		},
	}
	xmlBytes, xmlErr := s.Marshaller.Marshal(responseEnvelope)
	if xmlErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "could not marshal soap fault for: %s xmlError: %s\n", err, xmlErr)
		return
	}
	addSOAPHeader(w, len(xmlBytes), s.ContentType)
	w.Write(xmlBytes)
}

// WriteHeader first set the content-type header and then writes the header code.
func (s *Server) WriteHeader(w http.ResponseWriter, code int) {
	setContentType(w, s.ContentType)
	w.WriteHeader(code)
}

func setContentType(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
}

func addSOAPHeader(w http.ResponseWriter, contentLength int, contentType string) {
	setContentType(w, contentType)
	w.Header().Set("Content-Length", fmt.Sprint(contentLength))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	soapAction := r.Header.Get("SOAPAction")
	s.log(1, "Request", "ServeHTTP method:", r.Method, ", path:", r.URL.Path, ", SOAPAction", "\""+soapAction+"\"")
	// we have a valid request time to call the handler
	w = &responseWriter{
		logFunc:       s.Log,
		logLevel:      s.LogLevel,
		w:             w,
		outputStarted: false,
	}
	switch r.Method {
	case "POST":
		soapRequestBytes, err := ioutil.ReadAll(r.Body)
		// Our structs for Envelope, Header, Body and Fault are tagged with namespace for SOAP 1.1
		// Therefore we must adjust namespaces for incoming SOAP 1.2 messages
		if s.SoapVersion == SoapVersion12 {
			soapRequestBytes = replaceSoap12to11(soapRequestBytes)
		}

		if err != nil {
			s.handleError(fmt.Errorf("could not read POST:: %s", err), w)
			return
		}

		s.log(3, "Request", "request body:", string(soapRequestBytes))

		pathHandlers, ok := s.handlers[r.URL.Path]
		if !ok {
			s.log(1, "Request", "unknown path:", r.URL.Path)
			s.handleError(fmt.Errorf("unknown path %q", r.URL.Path), w)
			return
		}
		actionHandlers, ok := pathHandlers[soapAction]
		if !ok {
			s.log(1, "Request", "unknown action:", soapAction)
			s.handleError(fmt.Errorf("unknown action %q", soapAction), w)
			return
		}

		// we need to find out, what is in the body
		probeEnvelope := &Envelope{
			Body: Body{
				Content: &dummyContent{},
			},
		}

		if err := s.Marshaller.Unmarshal(soapRequestBytes, probeEnvelope); err != nil {
			s.handleError(fmt.Errorf("could not probe soap body content:: %s", err), w)
			return
		}
		t := probeEnvelope.Body.SOAPBodyContentType
		s.log(1, "Request", "found content type", t)
		actionHandler, ok := actionHandlers[t]
		if !ok {
			s.handleError(fmt.Errorf("no action handler for content type: %q", t), w)
			return
		}
		var request interface{}
		if actionHandler.requestFactory != nil {
			request = actionHandler.requestFactory()
		} else {
			request = reflect.New(reflect.TypeOf(actionHandler.requestStruct)).Interface()

		}

		envelope := &Envelope{
			Header: Header{},
			Body: Body{
				Content: request,
			},
		}

		if err := xml.Unmarshal(soapRequestBytes, &envelope); err != nil {
			s.handleError(fmt.Errorf("could not unmarshal request:: %s", err), w)
			return
		}
		s.log(2, "Request", "envelope:", s.jsonDump(envelope))

		var response interface{}
		if actionHandler.handlerWithInfo != nil {
			response, err = actionHandler.handlerWithInfo(request, w, r, actionHandler.handlerInfo)
		} else {
			response, err = actionHandler.handler(request, w, r)
		}

		if err != nil {
			s.log(1, "Request", "action handler threw up", err.Error())
			s.handleError(err, w)
			return
		}
		s.log(2, "Response", "msg", s.jsonDump(response))
		if !w.(*responseWriter).outputStarted {
			responseEnvelope := &Envelope{
				Body: Body{
					Content: response,
				},
			}
			xmlBytes, err := s.Marshaller.Marshal(responseEnvelope)
			// Adjust namespaces for SOAP 1.2
			if s.SoapVersion == SoapVersion12 {
				xmlBytes = replaceSoap11to12(xmlBytes)
			}
			if err != nil {
				s.handleError(fmt.Errorf("could not marshal response:: %s", err), w)
			}
			addSOAPHeader(w, len(xmlBytes), s.ContentType)
			w.Write(xmlBytes)
		} else {
			s.log(1, "Respose action handler sent its own output")
		}

	default:
		// this will be a soap fault !?
		s.handleError(errors.New("this is a soap service - you have to POST soap requests"), w)
	}
}

func (s *Server) jsonDump(v interface{}) string {
	if s.Log == nil {
		return "not dumping"
	}
	jsonBytes, err := json.MarshalIndent(v, "", "	")
	if err != nil {
		return "error in json dump :: " + err.Error()
	}
	return string(jsonBytes)
}
