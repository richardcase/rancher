package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"k8s.io/apiserver/pkg/server/healthz"
)

func Register(router *chi.Mux) {
	healthz.InstallHandler((*muxWrapper)(router))
	router.Handle("/ping", Pong())
}

func Pong() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("pong"))
	})
}

type muxWrapper chi.Mux

func (m *muxWrapper) Handle(path string, handler http.Handler) {
	(*chi.Mux)(m).Handle(path, handler)
}
