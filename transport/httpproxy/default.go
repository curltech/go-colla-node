package httpproxy

import (
	"github.com/vulcand/oxy/forward"
	"net/http"
	"net/url"
)

func Start() {
	// Forwards incoming requests to whatever location URL points to, adds proper forwarding headers
	fwd, _ := forward.New()

	redirect := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// let us forward this request to another server
		out, err := url.ParseRequestURI("http://localhost:63450")
		if err != nil {
			panic(err)
		}
		req.URL = out
		fwd.ServeHTTP(w, req)
	})

	// that's it! our reverse proxy is ready!
	s := &http.Server{
		Addr:    ":8080",
		Handler: redirect,
	}
	s.ListenAndServe()
}
