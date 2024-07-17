package sse

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	toolkitNet "gitlab.firecloud.wan/devops/ops-toolkit/net"
	"gopkg.in/antage/eventsource.v1"
)

func Sse(msg chan string, uri string, l net.Listener) {
	es := eventsource.New(nil, nil)
	defer es.Close()
	switch {
	case !strings.HasPrefix(uri, "/"):
		uri = "/" + uri
	case uri == "":
		uri = "/events"
	default:
		uri = "/events"
	}

	http.Handle(uri, es)
	go func() {
		id := 1
		for s := range msg {
			if s == "" {
				continue
			}
			id++
			es.SendEventMessage(s, "", strconv.Itoa(id))
		}
	}()
	//nolint:gosec
	if err := http.Serve(l, nil); err != nil {
		panic(err)
	}
}
func NewNetListen() (net.Listener, string, int) {
	address, err := toolkitNet.PhysicsCNIAddress()
	if err != nil {
		panic(err)
	}
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:0", address[0]))
	if err != nil {
		panic(err)
	}
	//nolint:forcetypeassert
	return l, address[0], l.Addr().(*net.TCPAddr).Port
}
