package main

import (
	"fmt"
	"time"
)

type A struct {
}

func (a *A) greet() {
	for {
		fmt.Println("Hello, playground")
		<-time.After(time.Second)
	}
}

func main() {
	c := make(chan (*A))
	go func() { c <- nil }()
	s, moar := <-c
	if s == nil {
		if moar {
			fmt.Println("moar")
		} else {
			fmt.Println("nomoar")
		}
	} else {
		fmt.Println("dafuq")
	}

	go func() { close(c) }()
	s, moar = <-c
	if s == nil {
		if moar {
			fmt.Println("moar")
		} else {
			fmt.Println("nomoar")
		}
	} else {
		fmt.Println("dafuq")
	}
}
