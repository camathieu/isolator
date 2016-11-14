package isolator

import (
	"net/http"
	"log"
	"io"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/root-gg/isolator/common"
	"io/ioutil"
	"sync"
)

const (
	IDLE = iota
	PROXY
	CLOSED
)

type ProxyConnection struct {
	pp *ProxyPool
	ws *websocket.Conn

	status int
	lock sync.Mutex

	nextResponse      chan(chan(io.Reader))
}

func NewProxyConnection(pp *ProxyPool, ws *websocket.Conn) (pc *ProxyConnection){
	pc = new(ProxyConnection)
	pc.pp = pp
	pc.ws = ws
	pc.nextResponse = make(chan(chan(io.Reader)))

	go pc.read()

	return
}

func (pc *ProxyConnection) read(){
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Websocket crash recovered : %s", r)
		}
		pc.Close()
	}()

	for {
		if pc.status == CLOSED {
			break
		}

		// https://godoc.org/github.com/gorilla/websocket#hdr-Control_Messages
		//
		// We need to ensure :
		//  - no concurrent calls to ws.NextReader() / ws.ReadMessage()
		//  - only one reader exists at a time
		//  - wait for reader to be consumed before requesting the next one
		//  - always be reading on the socket to be able to process control messages ( ping / pong / close )

		// We will block here until a message is received or the ws is closed
		_, reader, err := pc.ws.NextReader()
		if err != nil {
			pc.Close()
			return
		}

		if pc.status != PROXY {
			// We received a wild unexpected message
			break
		}

		// We received a message from the proxy
		// It is expected to be either a HttpResponse or a HttpResponseBody
		// We wait for proxyRequest to send a channel to get the message
		c := <-pc.nextResponse
		if c == nil {
			// We have been unlocked by Close()
			break
		}

		// Send the reader back to proxyRequest
		c <- reader

		// Wait for proxyRequest to close the channel
		// this notify that it is done with the reader
		<-c
	}
}

func (pc *ProxyConnection) proxyRequest(w http.ResponseWriter, r *http.Request) (err error){
	if pc.status != IDLE {
		return fmt.Errorf("Proxy connection is not READY")
	}
	pc.status = PROXY

	log.Printf("proxy request to %s", pc.pp.name)

	// Serialize request
	jsonReq, err := json.Marshal(common.SerializeHttpRequest(r))
	if err != nil {
		return fmt.Errorf("Unable to serialize request : %s",err)
	}

	log.Printf("write request")

	// Send serialized request to the proxy
	err = pc.ws.WriteMessage(websocket.TextMessage,jsonReq)
	if err != nil {
		return fmt.Errorf("Unable to write request : %s",err)
	}

	log.Printf("write request body")

	// Send the request body to the proxy
	bodyWriter, err := pc.ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return fmt.Errorf("Unable to get request body writer : %s",err)
	}
	_, err = io.Copy(bodyWriter,r.Body)
	if err != nil {
		return fmt.Errorf("Unable to pipe request body : %s",err)
	}

	err = bodyWriter.Close()
	if err != nil {
		return fmt.Errorf("Unable to pipe request body (close) : %s",err)
	}

	log.Printf("read response")

	// Send a new channel to read() to get the next message reader
	// which is expected to be the serialized response
	responseChannel := make(chan(io.Reader))
	pc.nextResponse <- responseChannel
	responseReader, more := <- responseChannel
	if responseReader == nil {
		if more {
			// If more is false the channel is already closed
			close(responseChannel)
		}
		return fmt.Errorf("Unable to get http response reader : %s",err)
	}

	jsonResponse, err := ioutil.ReadAll(responseReader)
	if err != nil {
		close(responseChannel)
		return fmt.Errorf("Unable to read http response : %s",err)
	}

	// Notify read() that we are done reading the serialized response
	close(responseChannel)

	// Unserialize response
	httpResponse := new(common.HttpResponse)
	err = json.Unmarshal(jsonResponse, httpResponse)
	if err != nil {
		return fmt.Errorf("Unable to unserialize http response : %s",err)
	}

	log.Printf("write response")

	// Write response headers to the client
	for header, values := range httpResponse.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.WriteHeader(httpResponse.StatusCode)

	log.Printf("read response body")

	// Send a new channel to read() to get the next message reader
	// which is expected to be the response body ( binary )
	responseBodyChannel := make(chan(io.Reader))
	pc.nextResponse <- responseBodyChannel
	responseBodyReader, more := <- responseBodyChannel
	if responseBodyReader == nil {
		if more {
			// If more is false the channel is already closed
			close(responseChannel)
		}
		return fmt.Errorf("Unable to get http response body reader : %s",err)
	}

	log.Printf("write response body")

	// Pipe the response body from the proxy to the client
	_, err = io.Copy(w,responseBodyReader)
	if err != nil {
		close(responseBodyChannel)
		return fmt.Errorf("Unable to pipe response body : %s",err)
	}

	// Notify read() that we are done reading the response body
	close(responseBodyChannel)

	log.Printf("yata")

	pc.status = IDLE
	return
}

func (pc *ProxyConnection) Close(){
	pc.lock.Lock()
	defer pc.lock.Unlock()

	if pc.status == CLOSED {
		return
	}

	log.Printf("Closing connection from %s", pc.pp.name)
	pc.ws.Close()

	// Unlock a possible read
	close(pc.nextResponse)
	pc.status = CLOSED
}