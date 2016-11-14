package isolator

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"io"
	"encoding/json"
	"fmt"
	"github.com/root-gg/isolator/common"
)

type Proxy struct {
	Hostname string
	ws *websocket.Conn
}

func NewProxy(hostname string, ws *websocket.Conn) (p *Proxy){
	p = new(Proxy)
	p.Hostname = hostname
	p.ws = ws
	return
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) (err error){
	log.Printf("proxy request to %s\n", p.Hostname)

	// Encode request
	jsonReq, err := json.Marshal(common.SerializeHttpRequest(r))
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to serialize request : %s\n",err),526)
		return
	}

	// Write request
	err = p.ws.WriteMessage(websocket.TextMessage,jsonReq)
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to write request : %s\n",err),526)
		return
	}

	// Pipe response body
	bodyWriter, err := p.ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to get request body writer : %s\n",err),526)
		return
	}
	_, err = io.Copy(bodyWriter,r.Body)
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to get pipe request body : %s\n",err),526)
		return
	}

	err = bodyWriter.Close()
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to get pipe request body (close) : %s\n",err),526)
		return
	}

	// Write response
	_, jsonResponse, err := p.ws.ReadMessage()
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to get response : %s\n",err),526)
		return
	}

	// Unserialize response
	httpResponse := new(common.HttpResponse)
	err = json.Unmarshal(jsonResponse, httpResponse)
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to unserialize http response : %s\n",err),526)
	}

	// Write response headers
	for header, values := range httpResponse.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.WriteHeader(httpResponse.StatusCode)

	// Write response body
	_, bodyReader, err := p.ws.NextReader()
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to get response body writer : %s\n",err),526)
		return
	}

	_, err = io.Copy(w,bodyReader)
	if err != nil {
		http.Error(w,fmt.Sprintf("Unable to pipe response body : %s\n",err),526)
		return
	}

	return
}