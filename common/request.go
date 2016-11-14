package common

import (
	"net/http"
)

type HttpRequest struct {
	Method           string
	URL              string
	Proto            string
	ProtoMajor       int
	ProtoMinor       int
	Header           http.Header
	ContentLength    int64
}

func SerializeHttpRequest(req *http.Request) *HttpRequest {
	r := new(HttpRequest)
	r.Method = req.Method
	r.Proto = req.Proto
	r.ProtoMajor = req.ProtoMajor
	r.ProtoMinor = req.ProtoMinor
	r.Header = req.Header
	r.ContentLength = req.ContentLength
	return r
}

func UnserializeHttpRequest(req *HttpRequest) *http.Request {
	r := new(http.Request)
	r.Method = req.Method
	r.Proto = req.Proto
	r.ProtoMajor = req.ProtoMajor
	r.ProtoMinor = req.ProtoMinor
	r.Header = req.Header
	r.ContentLength = req.ContentLength
	return r
}