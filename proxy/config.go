package proxy

type ProxyConfig struct {
	Name              string
	Targets           []string
	PoolIdleSize      int
	PoolConcurentSize int
	PoolMaxSize       int
}

func NewProxyConfig() (pc *ProxyConfig) {
	pc = new(ProxyConfig)
	pc.Targets = make([]string, 0)
	pc.PoolIdleSize = 10
	pc.PoolConcurentSize = 5
	pc.PoolMaxSize = 100
	return
}
