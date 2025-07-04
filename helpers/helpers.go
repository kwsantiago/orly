package helpers

import (
	"net/http"
	"strings"
)

func GenerateDescription(text string, scopes []string) string {
	if len(scopes) == 0 {
		return text
	}
	result := make([]string, 0)
	for _, value := range scopes {
		result = append(result, "`"+value+"`")
	}
	return text + "<br/><br/>**Scopes**<br/>" + strings.Join(result, ", ")
}

func GetRemoteFromReq(r *http.Request) (rr string) {
	// reverse proxy should populate this field so we see the remote not the
	// proxy
	remoteAddress := r.Header.Get("X-Forwarded-For")
	if remoteAddress == "" {
		remoteAddress = r.Header.Get("Forwarded")
		if remoteAddress == "" {
			rr = r.RemoteAddr
			return
		} else {
			splitted := strings.Split(remoteAddress, ", ")
			if len(splitted) >= 1 {
				forwarded := strings.Split(splitted[0], "=")
				if len(forwarded) == 2 {
					// by the standard this should be the address of the client.
					rr = splitted[1]
				}
				return
			}
		}
	}
	splitted := strings.Split(remoteAddress, " ")
	if len(splitted) == 1 {
		rr = splitted[0]
	}
	if len(splitted) == 2 {
		sp := strings.Split(splitted[0], ",")
		rr = sp[0]
	}
	return
}
