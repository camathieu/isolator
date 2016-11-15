package proxy

import (
	"log"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"net/url"
	"io/ioutil"
	"github.com/root-gg/isolator/common"
	"time"
	"sync"
)

const (
	CONNECTING = iota
	IDLE
	RUNNING
	CLOSED
)

type ProxyConnection struct {
	pool *ConnectionPool
	ws *websocket.Conn
	last time.Time
	status int
}

func NewProxyConnection(pool *ConnectionPool) (conn *ProxyConnection) {
	conn = new(ProxyConnection)
	conn.pool = pool
	conn.status = CONNECTING
	return
}

func (conn *ProxyConnection) Connect() (err error){
	log.Printf("Connecting to %s", conn.pool.target)

	conn.ws, _, err = conn.pool.proxy.dialer.Dial(conn.pool.target, nil)
	if err != nil {
		return err
	}

	conn.ws.SetCloseHandler(func(mt int, err string) error{
		log.Println("connection lost")
		conn.pool.Remove(conn)
		return nil
	})

	conn.last = time.Now()

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

	var wlock sync.Mutex

	//go func(){
	//	for {
	//		time.Sleep(time.Second)
	//		wlock.Lock()
	//		err = conn.ws.WriteControl(websocket.PingMessage,[]byte("ping"), time.Now().Add(time.Second))
	//		if err != nil {
	//			conn.Close()
	//		}
	//		wlock.Unlock()
	//	}
	//}()

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
		go conn.pool.Connect()

		wlock.Lock()

		// Unserialize request
		httpRequest := new(common.HttpRequest)
		err = json.Unmarshal(jsonRequest, httpRequest)
		if err != nil {
			log.Printf("Unable to unserialize http request : %s", err)
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
			log.Printf("Unable to parse URL : %v", err)
			break
		}
		req.URL = URL

		// Protect against trolls
		//req.Header.Del("X-PROXY-DESTINATION")

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
		err = conn.ws.WriteMessage(websocket.TextMessage,jsonResponse)
		if err != nil {
			log.Printf("Unable to write response : %v", err)
			break
		}

		log.Println("write response body")

		// Pipe response body
		bodyWriter, err := conn.ws.NextWriter(websocket.BinaryMessage)
		if (err != nil){
			log.Printf("Unable to get response body writer : %v",err)
			break
		}
		_, err = io.Copy(bodyWriter,resp.Body)
		if err != nil {
			log.Printf("Unable to get pipe response body : %v",err)
			break
		}
		bodyWriter.Close()

		wlock.Unlock()
	}
}

func (conn *ProxyConnection) Close() {
	defer conn.pool.Remove(conn)
	conn.status = CLOSED
	conn.ws.Close()
}
