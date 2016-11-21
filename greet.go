package main

import (
	"fmt"
	"time"
)

var id int = 0

type A struct {
	id int
}

func newA() (a *A) {
	a = new(A)
	a.id = id
	id++
	return
}

func (a *A) generate(c chan(string)) {
	i := 0
	for {
		c <- fmt.Sprintf("%d %d",a.id,i)
		i++
		if i == 10 {
			break
		}
	}
}

func main() {
	c := make(chan(string))
	for i:= 0 ; i < 3 ; i++{
		a := newA()
		go a.generate(c)
	}
	for s := range c {
		fmt.Println(s)
		time.Sleep(10 * time.Millisecond)
	}
}
