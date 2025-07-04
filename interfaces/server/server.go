package server

import (
	"net/http"
)

type I interface {
	HandleRelayInfo(w http.ResponseWriter, r *http.Request)
}
