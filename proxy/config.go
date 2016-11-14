package proxy

type ProxyConfig struct {
	Name string
	Targets []string
	PoolIdleSize int
	PoolMaxSize int
}

func NewProxyConfig() (pc *ProxyConfig){
	pc = new(ProxyConfig)
	pc.Targets = make([]string,0)
	pc.PoolIdleSize = 10
	pc.PoolMaxSize = 100
	return
}