package multiclustermanager

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rancher/rancher/pkg/api/norman"
	"github.com/rancher/rancher/pkg/api/norman/customization/aks"
	"github.com/rancher/rancher/pkg/api/norman/customization/clusterregistrationtokens"
	"github.com/rancher/rancher/pkg/api/norman/customization/gke"
	"github.com/rancher/rancher/pkg/api/norman/customization/oci"
	"github.com/rancher/rancher/pkg/api/norman/customization/vsphere"
	managementapi "github.com/rancher/rancher/pkg/api/norman/server"
	"github.com/rancher/rancher/pkg/api/steve/supportconfigs"
	"github.com/rancher/rancher/pkg/auth/providers/publicapi"
	"github.com/rancher/rancher/pkg/auth/providers/saml"
	"github.com/rancher/rancher/pkg/auth/requests"
	"github.com/rancher/rancher/pkg/auth/requests/sar"
	"github.com/rancher/rancher/pkg/auth/tokens"
	"github.com/rancher/rancher/pkg/auth/webhook"
	"github.com/rancher/rancher/pkg/channelserver"
	"github.com/rancher/rancher/pkg/clustermanager"
	rancherdialer "github.com/rancher/rancher/pkg/dialer"
	"github.com/rancher/rancher/pkg/httpproxy"
	k8sProxyPkg "github.com/rancher/rancher/pkg/k8sproxy"
	"github.com/rancher/rancher/pkg/metrics"
	"github.com/rancher/rancher/pkg/multiclustermanager/whitelist"
	"github.com/rancher/rancher/pkg/rbac"
	"github.com/rancher/rancher/pkg/rkenodeconfigserver"
	"github.com/rancher/rancher/pkg/telemetry"
	"github.com/rancher/rancher/pkg/tunnelserver/mcmauthorizer"
	"github.com/rancher/rancher/pkg/types/config"
	"github.com/rancher/rancher/pkg/version"
	"github.com/rancher/steve/pkg/auth"
)

func router(ctx context.Context, localClusterEnabled bool, tunnelAuthorizer *mcmauthorizer.Authorizer, scaledContext *config.ScaledContext, clusterManager *clustermanager.Manager) (func(http.Handler) http.Handler, error) {
	var (
		k8sProxy             = k8sProxyPkg.New(scaledContext, scaledContext.Dialer, clusterManager)
		connectHandler       = scaledContext.Dialer.(*rancherdialer.Factory).TunnelServer
		connectConfigHandler = rkenodeconfigserver.Handler(tunnelAuthorizer, scaledContext)
		clusterImport        = clusterregistrationtokens.ClusterImport{Clusters: scaledContext.Management.Clusters("")}
	)

	tokenAPI, err := tokens.NewAPIHandler(ctx, scaledContext, norman.ConfigureAPIUI)
	if err != nil {
		return nil, err
	}

	publicAPI, err := publicapi.NewHandler(ctx, scaledContext, norman.ConfigureAPIUI)
	if err != nil {
		return nil, err
	}

	managementAPI, err := managementapi.New(ctx, scaledContext, clusterManager, k8sProxy, localClusterEnabled)
	if err != nil {
		return nil, err
	}

	metaProxy, err := httpproxy.NewProxy("/proxy/", whitelist.Proxy.Get, scaledContext)
	if err != nil {
		return nil, err
	}

	metricsHandler := metrics.NewMetricsHandler(scaledContext, clusterManager, promhttp.Handler())

	channelserver := channelserver.NewHandler(ctx)

	supportConfigGenerator := supportconfigs.NewHandler(scaledContext)
	// Unauthenticated routes
	unauthed := chi.NewRouter()
	unauthed.Use(middleware.RequestID)
	unauthed.Use(middleware.RealIP)
	//unauthed.Use(middleware.Logger)
	//unauthed.Use(middleware.Recoverer)
	//TODO: MUX
	//unauthed.UseEncodedPath()

	//TODO: mux
	//unauthed.Use() Path("/").MatcherFunc(parse.MatchNotBrowser).Handler(managementAPI)
	//unauthed.With(NotForBrowser).Handle("/", managementAPI)
	unauthed.Handle("/", managementAPI)
	unauthed.Handle("/v3/connect/config", connectConfigHandler)
	unauthed.Handle("/v3/connect", connectHandler)
	unauthed.Handle("/v3/connect/register", connectHandler)
	unauthed.Handle("/v3/import/{token}_{clusterId}.yaml", http.HandlerFunc(clusterImport.ClusterImportHandler))
	unauthed.Method("GET", "/v3/settings/cacerts", managementAPI)
	unauthed.Method("GET", "/v3/settings/first-login", managementAPI)
	unauthed.Method("GET", "/v3/settings/ui-banners", managementAPI)
	unauthed.Method("GET", "/v3/settings/ui-issues", managementAPI)
	unauthed.Method("GET", "/v3/settings/ui-pl", managementAPI)
	unauthed.Method("GET", "/v3/settings/ui-brand", managementAPI)
	unauthed.Method("GET", "/v3/settings/ui-default-landing", managementAPI)
	unauthed.Handle("/rancherversion", version.NewVersionHandler())
	unauthed.Handle("/v1-{prefix}-release/channel", channelserver)
	unauthed.Handle("/v1-{prefix}-release/release", channelserver)
	unauthed.Handle("/v1-saml", saml.AuthHandler())
	unauthed.Handle("/v3-public", publicAPI)

	// Authenticated routes
	authed := chi.NewRouter()
	//authed.Use(middleware.RequestID)
	//authed.Use(middleware.RealIP)
	//authed.Use(middleware.Logger)
	//authed.Use(middleware.Recoverer)
	//authed.UseEncodedPath()
	impersonatingAuth := auth.ToMiddleware(requests.NewImpersonatingAuth(sar.NewSubjectAccessReview(clusterManager)))
	accessControlHandler := rbac.NewAccessControlHandler()

	authed.Use(impersonatingAuth)
	authed.Use(accessControlHandler)
	authed.Use(requests.NewAuthenticatedFilter)

	authed.Handle("/meta/{resource:aks.+}", aks.NewAKSHandler(scaledContext))
	authed.Handle("/meta/{resource:gke.+}", gke.NewGKEHandler(scaledContext))
	authed.Handle("/meta/oci/{resource}", oci.NewOCIHandler(scaledContext))
	authed.Handle("/meta/vsphere/{field}", vsphere.NewVsphereHandler(scaledContext))
	authed.Method("POST", "/v3/tokenreview", &webhook.TokenReviewer{})
	authed.Handle("/metrics/{clusterID}", metricsHandler)
	authed.Handle(supportconfigs.Endpoint, &supportConfigGenerator)
	authed.Handle("/k8s/clusters/", k8sProxy)
	authed.Handle("/meta/proxy", metaProxy)
	authed.Handle("/v1-telemetry", telemetry.NewProxy())
	authed.Handle("/v3/identit", tokenAPI)
	authed.Handle("/v3/token", tokenAPI)
	authed.Handle("/v3", managementAPI)

	// Metrics authenticated route
	metricsAuthed := chi.NewRouter()
	metricsAuthed.Use(middleware.RequestID)
	metricsAuthed.Use(middleware.RealIP)
	metricsAuthed.Use(middleware.Logger)
	metricsAuthed.Use(middleware.Recoverer)
	//TODO: mux
	//metricsAuthed.UseEncodedPath()
	tokenReviewAuth := auth.ToMiddleware(requests.NewTokenReviewAuth(scaledContext.K8sClient.AuthenticationV1()))
	metricsAuthed.Use(tokenReviewAuth.Chain(impersonatingAuth))
	metricsAuthed.Use(accessControlHandler)
	metricsAuthed.Use(requests.NewAuthenticatedFilter)

	metricsAuthed.Handle("/metrics", metricsHandler)

	//unauthed.NotFoundHandler = authed
	unauthed.NotFound(authed.ServeHTTP)
	//authed.NotFoundHandler = metricsAuthed
	authed.NotFound(metricsAuthed.ServeHTTP)
	return func(next http.Handler) http.Handler {
		metricsAuthed.NotFound(next.ServeHTTP)
		return unauthed
	}, nil
}

func IsBrowser(req *http.Request, checkAccepts bool) bool {
	accepts := strings.ToLower(req.Header.Get("Accept"))
	userAgent := strings.ToLower(req.Header.Get("User-Agent"))

	if accepts == "" || !checkAccepts {
		accepts = "*/*"
	}

	// User agent has Mozilla and browser accepts */*
	return strings.Contains(userAgent, "mozilla") && strings.Contains(accepts, "*/*")
}

func ForBrowser(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		isBrowser := IsBrowser(r, true)
		if isBrowser {
			next.ServeHTTP(w, r)
		}

	}

	return http.HandlerFunc(fn)
}

func NotForBrowser(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		isBrowser := IsBrowser(r, true)
		if !isBrowser {
			next.ServeHTTP(w, r)
		}

	}

	return http.HandlerFunc(fn)
}
