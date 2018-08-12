package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/go-systemd/activation"
	"github.com/hashicorp/consul/api"
)

type ctxKey int

const (
	consulClient ctxKey = iota
)

func WithConsulClient(ctx context.Context, cli *api.Client) context.Context {
	return context.WithValue(ctx, consulClient, cli)
}

func GetConsulClient(ctx context.Context) *api.Client {
	return ctx.Value(consulClient).(*api.Client)
}

func CopyHeaders(dst, src http.Header) {
	for k, v := range src {
		for _, e := range v {
			dst.Add(k, e)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	client := GetConsulClient(r.Context())
	agent := client.Agent()
	services, err := agent.Services()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hostParts := strings.Split(r.Host, ".")
	service, ok := services[hostParts[0]]
	if !ok {
		http.Error(w, "Service "+hostParts[0]+" not found", http.StatusNotFound)
		return
	}

	backendURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", service.Address, service.Port),
	}
	proxy := httputil.NewSingleHostReverseProxy(backendURL)
	proxy.ServeHTTP(w, r)
}

func main() {
	var l net.Listener
	consulConfig := api.DefaultConfig()
	client, err := api.NewClient(consulConfig)
	if err != nil {
		log.Fatal("Connecting to Consul:", err)
	}

	passedSockets, err := activation.Listeners()
	if err != nil {
		log.Fatal("Fetching passed sockets: ", err)
	}

	proto := "tcp"
	addr := ":8080"

	switch len(os.Args) {
	case 0, 1:
	case 2:
		addr = os.Args[1]
	case 3:
		proto = os.Args[1]
		addr = os.Args[2]
	default:
		log.Fatal("Too many arguments - expected [[tcp] addr:port]")
	}

	switch len(passedSockets) {
	case 0:
		log.Println("No passed file descriptor - binding to", proto, addr)
		l, err = net.Listen(proto, addr)
		if err != nil {
			log.Fatal("Listening on ", proto, addr, ":", err)
		}

	default:
		l = passedSockets[0]
		for i := 1; i < len(passedSockets); i++ {
			if err := passedSockets[i].Close(); err != nil {
				log.Fatal("Closing passed socket:", err)
			}
		}
	}

	server := http.Server{
		Addr: l.Addr().String(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(WithConsulClient(r.Context(), client))
			handler(w, r)
		}),
	}

	err = server.Serve(l)
	if err != nil {
		panic(err)
	}
}
