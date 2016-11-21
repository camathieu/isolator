package proxy

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/root-gg/isolator/common"
	"fmt"
)

const (
	CONNECTING = iota
	IDLE
	RUNNING
	CLOSED
)

// ProxyConection handle a single websocket (HTTP/TCP) connection to an IzolatorServer
type ProxyConnection struct {
	pool   *ConnectionPool
	ws     *websocket.Conn
	last   time.Time
	status int
}

// NewProxyConnection create a ProxyConnection object
func NewProxyConnection(pool *ConnectionPool) (conn *ProxyConnection) {
	conn = new(ProxyConnection)
	conn.pool = pool
	conn.status = CONNECTING
	return
}

// Connect to the IsolatorServer via websocket
func (conn *ProxyConnection) Connect() (err error) {
	log.Printf("Connecting to %s", conn.pool.target)

	// Create a new TCP(/TLS) connection ( no use of net.http )
	conn.ws, _, err = conn.pool.proxy.dialer.Dial(conn.pool.target, nil)
	if err != nil {
		return err
	}

	// Send the greeting message with proxy id and wanted pool size.
	greeting := fmt.Sprintf("%s_%d", conn.pool.proxy.config.Name,conn.pool.proxy.config.PoolIdleSize)
	err = conn.ws.WriteMessage(websocket.TextMessage, []byte(greeting));
	if err != nil {
		log.Println("greeting error :", err)
		conn.Close()
		return
	}

	conn.last = time.Now()

	go conn.Serve()

	return
}

// Serve is the main loop it :
//  - wait to receive HTTP requests from the IzolatorServer
//  - execute HTTP requests
//  - send HTTP response back to the IzolatorServer
//
// As in the server code there is no buffering of HTTP request/response body
// As is the server if any error occurs the connection is closed/throwed
func (conn *ProxyConnection) Serve() {
	defer conn.Close()


	// Keep connection alive
	go func(){
		for {
			time.Sleep(30 * time.Second)
			err := conn.ws.WriteControl(websocket.PingMessage,[]byte{}, time.Now().Add(time.Second))
			if err != nil {
				conn.Close()
			}
		}
	}()

	for {
		// Read request
		conn.status = IDLE
		_, jsonRequest, err := conn.ws.ReadMessage()
		if err != nil {
			log.Println("Unable to read request", err)
			break
		}

		log.Printf("got request : %s", jsonRequest)

		//if string(jsonRequest) == "ruok" {
		//	err = conn.ws.WriteMessage(websocket.TextMessage,[]byte("imok"))
		//	if err != nil {
		//		log.Println("Unable respond to keepalive request", err)
		//		break
		//	}
		//	continue
		//}

		conn.status = RUNNING
		conn.last = time.Now()

		// Trigger a pool refresh to open new connections if needed
		go conn.pool.connect()

		// Unserialize request
		httpRequest := new(common.HttpRequest)
		err = json.Unmarshal(jsonRequest, httpRequest)
		if err != nil {
			log.Printf("Unable to unserialize http request : %s", err)
			break
		}
		req, err := common.UnserializeHttpRequest(httpRequest)
		if err != nil {
			log.Printf("Unable to unserialize http request : %v", err)
			break
		}

		// Pipe request body
		_, bodyReader, err := conn.ws.NextReader()
		if err != nil {
			log.Printf("Unable to get response body reader : %v", err)
			break
		}
		req.Body = ioutil.NopCloser(bodyReader)

		// Execute request
		log.Printf("execute request")
		resp, err := conn.pool.proxy.client.Do(req)
		if err != nil {
			log.Printf("Unable to execute request : %v", err)
			break
		}

		// Serialize response
		jsonResponse, err := json.Marshal(common.SerializeHttpResponse(resp))
		if err != nil {
			log.Printf("Unable to serialize response : %v", err)
			break
		}

		log.Println("write response")

		// Write response
		err = conn.ws.WriteMessage(websocket.TextMessage, jsonResponse)
		if err != nil {
			log.Printf("Unable to write response : %v", err)
			break
		}

		log.Println("write response body")

		// Pipe response body
		bodyWriter, err := conn.ws.NextWriter(websocket.BinaryMessage)
		if err != nil {
			log.Printf("Unable to get response body writer : %v", err)
			break
		}
		_, err = io.Copy(bodyWriter, resp.Body)
		if err != nil {
			log.Printf("Unable to get pipe response body : %v", err)
			break
		}
		bodyWriter.Close()
	}
}

// Close close the connection and remove it from the pool
func (conn *ProxyConnection) Close() {
	defer conn.pool.Remove(conn)
	conn.status = CLOSED
	conn.ws.Close()
}
