package isolator

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"time"
)

type Isolator struct {
	pool *ProxyPool
	upgrader websocket.Upgrader
}

func NewIsolator() (i *Isolator) {
	i = new(Isolator)
	i.pool = NewProxyPool()
	i.upgrader = websocket.Upgrader{}
	return
}

func (i *Isolator) Start() {
	r := http.NewServeMux()
	r.HandleFunc("/request", i.request)
	r.HandleFunc("/proxy", i.proxy)

	s := &http.Server{Addr: "127.0.0.1:8080", Handler: r}
	log.Fatal(s.ListenAndServe())
}

func (i *Isolator) request(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v %v",r.Method,r.Header)

	p,err := i.pool.TakeWithTimeout(time.Second)
	if (err != nil){
		http.Error(w,"Unable to get a proxy connection",500)
	}

	err = p.handle(w,r)
	if (err == nil){
		i.pool.Offer(p)
	}
}

func (i *Isolator) proxy(w http.ResponseWriter, r *http.Request) {
	ws, err := i.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade proxy error :", err)
		return
	}

	ws.SetCloseHandler(func(code int, text string) (err error) { log.Println("close") ; return })

	mt, hostname, err := ws.ReadMessage()
	if err != nil {
		log.Println("Greeting error :", err)
		return
	}

	if (mt != websocket.TextMessage){
		log.Printf("Greeting error : invalid message type %v expected %v\n",mt,websocket.TextMessage)
		return
	}

	p := NewProxy(string(hostname),ws)
	log.Printf("New proxy %s registered\n",p.Hostname)
	i.pool.Offer(p)
}