package pprof

import (
	"log"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof" //nolint:gosec
)

func Pprof(addr string) {
	addrTmp := addr
	if addrTmp == "" {
		addrTmp = ":6060"
	}
	//nolint:gosec
	err := http.ListenAndServe(addrTmp, nil)
	if err != nil {
		log.Fatal(err)
	}
}
func RegisterHandlers(h func(pattern string, handler func(http.ResponseWriter, *http.Request))) {
	h("/debug/pprof/", pprof.Index)
	h("/debug/pprof/cmdline", pprof.Cmdline)
	h("/debug/pprof/profile", pprof.Profile)
	h("/debug/pprof/symbol", pprof.Symbol)
	h("/debug/pprof/trace", pprof.Trace)
}
