// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"log"
	"net/http"
	"io/ioutil"
)

var addr = flag.String("addr", "localhost:8081", "http service address")

func hello(w http.ResponseWriter, r *http.Request) {
	log.Println("hello")
	w.Write([]byte("hello world\n"))
}

func header(w http.ResponseWriter, r *http.Request) {
	log.Println("header")
	w.Header().Add("hello","world")
	w.Write([]byte("hello world in header\n"))
}

func post(w http.ResponseWriter, r *http.Request) {
	body,err := ioutil.ReadAll(r.Body)
	if (err != nil){
		log.Println(err)
	}
	r.Body.Close()

	log.Println("post")
	log.Println(string(body))

	w.Write(body)
}

func fail(w http.ResponseWriter, r *http.Request) {
	http.Error(w,"GO FUNK YOURSELF",666)
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/header", header)
	http.HandleFunc("/fail", fail)
	http.HandleFunc("/post", post)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
