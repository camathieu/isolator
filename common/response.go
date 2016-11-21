package common

import (
	"net/http"
)

// HttpResponse is a serializable version of http.Response ( with only useful fields )
type HttpResponse struct {
	Status        string
	StatusCode    int
//	Proto         string
//	ProtoMajor    int
//	ProtoMinor    int
	Header        http.Header
	ContentLength int64
}

// SerializeHttpResponse create a new HttpResponse from a http.Response
func SerializeHttpResponse(resp *http.Response) *HttpResponse {
	r := new(HttpResponse)
	r.Status = resp.Status
	r.StatusCode = resp.StatusCode
//	r.Proto = resp.Proto
//	r.ProtoMajor = resp.ProtoMajor
//	r.ProtoMinor = resp.ProtoMinor
	r.Header = resp.Header
	r.ContentLength = resp.ContentLength
	return r
}
