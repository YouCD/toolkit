package sse

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/r3labs/sse/v2"
	toolkitNet "github.com/youcd/toolkit/net"
)

func Sse(msg chan string, uri, streamID string, l net.Listener) {
	server := sse.New()
	server.CreateStream(streamID)
	mux := http.NewServeMux()

	defer server.Close()
	switch {
	case !strings.HasPrefix(uri, "/"):
		uri = "/" + uri
	case uri == "":
		uri = "/events"
	default:
		uri = "/events"
	}

	mux.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		go func() {
			<-r.Context().Done()
			fmt.Println("The client is disconnected here")
		}()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		server.ServeHTTP(w, r)
	})

	go func() {
		id := 1
		for s := range msg {
			if s == "" {
				continue
			}
			id++
			server.Publish(streamID, &sse.Event{
				Data: []byte(s),
				ID:   []byte(strconv.Itoa(id)),
			})
		}
	}()
	//nolint:gosec
	if err := http.Serve(l, mux); err != nil {
		panic(err)
	}
}
func NewNetListen(port int) (net.Listener, string, int) {
	address, err := toolkitNet.PhysicsCNIAddress()
	if err != nil {
		panic(err)
	}
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", address[0], port))
	if err != nil {
		panic(err)
	}
	//nolint:forcetypeassert
	return l, address[0], l.Addr().(*net.TCPAddr).Port
}
