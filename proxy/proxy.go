package proxy

import (
	"net/http"
	"crypto/tls"
	"github.com/gorilla/websocket"
)

type IzolatorProxy struct {
	config *ProxyConfig
	client *http.Client
	dialer *websocket.Dialer
	pools map[string]*ConnectionPool
}

func NewIzolatorProxy(config *ProxyConfig) (ip *IzolatorProxy){
	ip = new(IzolatorProxy)
	ip.config = config
	ip.client = &http.Client{}
	ip.dialer = &websocket.Dialer{ TLSClientConfig: &tls.Config{InsecureSkipVerify:true}}
	ip.pools = make(map[string]*ConnectionPool)
	return
}

func (ip *IzolatorProxy) Start() {
	for _, target := range ip.config.Targets {
		pool := NewConnectionPool(ip,target)
		ip.pools[target] = pool
		go pool.Start()
	}
}

func (ip *IzolatorProxy) Shutdown() {
	for _, pool := range ip.pools {
		pool.Shutdown()
	}
}