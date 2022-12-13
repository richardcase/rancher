package steve

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rancher/rancher/pkg/api/steve/aggregation"
	"github.com/rancher/rancher/pkg/api/steve/github"
	"github.com/rancher/rancher/pkg/api/steve/health"
	"github.com/rancher/rancher/pkg/api/steve/projects"
	"github.com/rancher/rancher/pkg/api/steve/proxy"
	"github.com/rancher/rancher/pkg/features"
	"github.com/rancher/rancher/pkg/provisioningv2/rke2/configserver"
	"github.com/rancher/rancher/pkg/provisioningv2/rke2/installer"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/rancher/pkg/wrangler"
	steve "github.com/rancher/steve/pkg/server"
)

func AdditionalAPIsPreMCM(config *wrangler.Context) func(http.Handler) http.Handler {
	if features.RKE2.Enabled() {
		connectHandler := configserver.New(config)
		mux := chi.NewRouter()
		mux.Use(middleware.RequestID)
		mux.Use(middleware.RealIP)
		mux.Use(middleware.Logger)
		mux.Use(middleware.Recoverer)

		//TODO: MUX
		//mux.UseEncodedPath()
		mux.Handle(configserver.ConnectAgent, connectHandler)
		mux.Handle(configserver.ConnectConfigYamlPath, connectHandler)
		mux.Handle(configserver.ConnectClusterInfo, connectHandler)
		mux.Handle(installer.SystemAgentInstallPath, installer.Handler)
		mux.Handle(installer.WindowsRke2InstallPath, installer.Handler)
		return func(next http.Handler) http.Handler {
			mux.NotFound(next.ServeHTTP)
			return mux
		}
	}

	return func(next http.Handler) http.Handler {
		return next
	}
}

func AdditionalAPIs(ctx context.Context, config *wrangler.Context, steve *steve.Server) (func(http.Handler) http.Handler, error) {
	clusterAPI, err := projects.Projects(ctx, steve)
	if err != nil {
		return nil, err
	}

	githubHandler, err := github.NewProxy(config.Core.Secret().Cache(),
		settings.GithubProxyAPIURL.Get(),
		"cattle-system",
		"github")
	if err != nil {
		return nil, err
	}

	mux := chi.NewRouter()
	//mux.Use(middleware.RequestID)
	//mux.Use(middleware.RealIP)
	//mux.Use(middleware.Logger)
	//mux.Use(middleware.Recoverer)

	//TODO: MUX
	//mux.UseEncodedPath()
	mux.Handle("/v1/github{path:.*}", githubHandler)
	mux.Handle("/v3/connect", Tunnel(config))
	health.Register(mux)

	return func(next http.Handler) http.Handler {
		mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
			clusterAPI(next).ServeHTTP(w, r)
		})
		return mux
	}, nil
}

func Tunnel(config *wrangler.Context) http.Handler {
	config.TunnelAuthorizer.Add(proxy.NewAuthorizer(config))
	config.TunnelAuthorizer.Add(aggregation.New(config))
	return config.TunnelServer
}
