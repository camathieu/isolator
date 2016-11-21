package common

type GreetingMessage struct {
	Id string
	PoolSize int
}

func NewGreetingMessage(id string, size int) (gm *GreetingMessage){
	gm = new(GreetingMessage)
	gm.Id = id
	gm.PoolSize = size
	return
}