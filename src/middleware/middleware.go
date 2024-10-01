package middleware

import (
	"log"
	"net/http"
)

// Outputs a log message for each request recieved in the format:
// (copied and tweaked from main backend)
// "[TIME] [DATE] INC REQ: [METHOD] [PATH] at [TIME] from [IP] response: [STATUS_CODE]"
func LogResultOfRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("INC REQ: %s --> %-7s %-30s --> %s",
		r.RemoteAddr,
		r.Method,
		r.URL.Path,
		w.Header().Get("Status-Code"),
	)
}
