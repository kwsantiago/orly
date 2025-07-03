package servemux

import (
	"net/http"
	"not.realy.lol/lol"
)

type S struct {
	*http.ServeMux
}

func New() (c *S) {
	lol.Tracer("New")
	c = &S{http.NewServeMux()}
	return
}

func (c *S) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lol.Tracer("ServeHTTP")
	defer func() { lol.Tracer("end ServeHTTP") }()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		return
	}
	c.ServeMux.ServeHTTP(w, r)
}
