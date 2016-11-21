package common

import (
	"net/http"
	"net/url"
)

// HttpRequest is a serializable version of http.Request ( with only usefull fields )
type HttpRequest struct {
	Method        string
	URL           string
	Proto         string
	ProtoMajor    int
	ProtoMinor    int
	Header        http.Header
	ContentLength int64
}

// SerializeHttpRequest create a new HttpRequest from a http.Request
func SerializeHttpRequest(req *http.Request) (r *HttpRequest) {
	r = new(HttpRequest)
	r.URL = req.URL.String()
	r.Method = req.Method
	r.Proto = req.Proto
	r.ProtoMajor = req.ProtoMajor
	r.ProtoMinor = req.ProtoMinor
	r.Header = req.Header
	r.ContentLength = req.ContentLength
	return
}

// SerializeHttpRequest create a new http.Request from a HttpRequest
func UnserializeHttpRequest(req *HttpRequest) (r *http.Request, err error) {
	r = new(http.Request)
	r.Method = req.Method
	r.URL, err = url.Parse(req.URL)
	if err != nil {
		return
	}
	r.Proto = req.Proto
	r.ProtoMajor = req.ProtoMajor
	r.ProtoMinor = req.ProtoMinor
	r.Header = req.Header
	r.ContentLength = req.ContentLength
	return
}
