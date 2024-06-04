package pprof

import (
	"log"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
)

func Pprof(addr string) {
	addrTmp := addr
	if addrTmp == "" {
		addrTmp = ":6060"
	}
	//nolint:gosec
	if err := http.ListenAndServe(addrTmp, nil); err != nil {
		log.Fatal(err)
	}
}
