package proxy

import (
	"github.com/gorilla/websocket"
	"sync"
	"log"
	"io/ioutil"
	"encoding/json"
	"io"
	"net/url"
	"github.com/root-gg/isolator/common"
	"time"
)

type ProxyConnection struct {
	pool *ConnectionPool
	ws *websocket.Conn
	last int64
	idle bool
}

func NewProxyConnection(pool *ConnectionPool) (conn *ProxyConnection) {
	conn = new(ProxyConnection)
	conn.pool = pool
	conn.idle = true
	return
}

func (conn *ProxyConnection) Connect() (err error){
	log.Printf("Connecting to %s\n", conn.pool.target)

	conn.ws, _, err = websocket.DefaultDialer.Dial(conn.pool.target, nil)
	if err != nil {
		return err
	}

	go conn.Serve()

	return
}

func (conn *ProxyConnection) Serve() {
	// defer remove conn
	defer conn.Close()

	// Greeting
	err := conn.ws.WriteMessage(websocket.TextMessage, []byte(conn.pool.proxy.config.Name))
	if err != nil {
		log.Println("greeting error :", err)
		return
	}

	for {
		// Read request
		_, jsonRequest, err := conn.ws.ReadMessage()
		if err != nil {
			log.Println("Unable to read request", err)
			break
		}
		log.Printf("request : %s", jsonRequest)

		conn.idle = false
		go conn.pool.Connect()

		// Unserialize request
		httpRequest := new(common.HttpRequest)
		err = json.Unmarshal(jsonRequest, httpRequest)
		if err != nil {
			log.Printf("Unable to unserialize http request : %s\n", err)
			break
		}
		req := common.UnserializeHttpRequest(httpRequest)

		dstURL := httpRequest.Header.Get("X-PROXY-DESTINATION")
		if dstURL == "" {
			log.Println("Missing X-PROXY-DESTINATION header")
			break
		}

		URL, err := url.Parse(dstURL)
		if err != nil {
			log.Printf("Unable to parse URL : %v\n", err)
			break
		}
		req.URL = URL

		// Pipe request body
		_, bodyReader, err := conn.ws.NextReader()
		if err != nil {
			log.Printf("Unable to get response body reader : %v\n", err)
			break
		}
		req.Body = ioutil.NopCloser(bodyReader)

		// Execute request
		log.Printf("executing req : %v\n", req)
		resp, err := conn.pool.proxy.client.Do(req)
		if err != nil {
			log.Printf("Unable to execute request : %v\n", err)
			break
		}

		// Serialize response
		jsonResponse, err := json.Marshal(common.SerializeHttpResponse(resp))
		if err != nil {
			log.Printf("Unable to serialize response : %v\n", err)
			break
		}

		// Write response
		err = conn.ws.WriteMessage(websocket.TextMessage,jsonResponse)
		if err != nil {
			log.Printf("Unable to write response : %v\n", err)
			return
			break
		}

		// Pipe response body
		bodyWriter, err := conn.ws.NextWriter(websocket.BinaryMessage)
		if (err != nil){
			log.Printf("Unable to get response body writer : %v\n",err)
			break
		}
		_, err = io.Copy(bodyWriter,resp.Body)
		bodyWriter.Close()

		conn.idle = true
	}
}

func (conn *ProxyConnection) Close() {
	defer conn.ws.Close()
	conn.idle = false
	conn.pool.Remove(conn)
}

type ConnectionPool struct {
	proxy *Proxy
	target string

	connections []*ProxyConnection
	lock sync.Mutex
}

func NewConnectionPool(proxy *Proxy, target string) (cp *ConnectionPool){
	cp = new(ConnectionPool)
	cp.proxy = proxy
	cp.target = target
	cp.connections = make([]*ProxyConnection,0)
	return
}

func (cp *ConnectionPool) Start() {
	cp.Connect()

	go func() {
		for {
			select {
			case <- time.Tick(time.Second):
				cp.GarbageCollector()
			}
		}
	}()
}

func (cp *ConnectionPool) Connect() {
	idle, total := cp.Size()

	if (idle >= cp.proxy.config.PoolIdleSize || total >= cp.proxy.config.PoolMaxSize) {
		return
	}

	log.Printf("Idle %v, Total %v",idle,total)

	conn := NewProxyConnection(cp)
	err := conn.Connect()
	if err == nil {
		cp.Offer(conn)
	}
}

func (cp *ConnectionPool) GarbageCollector() {
	cp.Connect()
}

func (cp *ConnectionPool) Size() (idle int, total int){
	cp.lock.Lock()
	defer cp.lock.Unlock()

	total = len(cp.connections)
	for _, connection :=  range cp.connections {
		if connection.idle {
			idle++
		}
	}

	return
}

func (cp *ConnectionPool) Offer(conn *ProxyConnection){
	cp.lock.Lock()
	defer cp.lock.Unlock()

	cp.connections = append(cp.connections,conn)

}

func (cp *ConnectionPool) Remove(conn *ProxyConnection){
	cp.lock.Lock()
	defer cp.lock.Unlock()

	// TODO Slice trick
}