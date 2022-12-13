package ui

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rancher/apiserver/pkg/parse"
	"github.com/rancher/rancher/pkg/cacerts"
	v3 "github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io/v3"
)

func New(_ v3.PreferenceCache, clusterRegistrationTokenCache v3.ClusterRegistrationTokenCache) http.Handler {
	router := chi.NewRouter()
	//router.Use(middleware.RequestID)
	//router.Use(middleware.RealIP)
	//router.Use(middleware.Logger)
	//router.Use(middleware.Recoverer)
	//TODO: mux
	//router.UseEncodedPath()

	router.Handle("/", PreferredIndex())
	router.Handle("/cacerts", cacerts.Handler(clusterRegistrationTokenCache))
	router.Handle("/asset-manifest.json", ember.ServeAsset())
	router.Handle("/crossdomain.xml", ember.ServeAsset())
	router.Handle("/dashboard", http.RedirectHandler("/dashboard/", http.StatusFound))
	router.Handle("/dashboard/", vue.IndexFile())
	router.Handle("/humans.txt", ember.ServeAsset())
	router.Handle("/index.txt", ember.ServeAsset())
	router.Handle("/robots.txt", ember.ServeAsset())
	router.Handle("/VERSION.txt", ember.ServeAsset())
	router.Handle("/favicon.png", vue.ServeFaviconDashboard())
	router.Handle("/favicon.ico", vue.ServeFaviconDashboard())
	router.HandleFunc("/verify-auth-azure?state={state}", redirectAuth)
	router.HandleFunc("/verify-auth?state={state}", redirectAuth)
	router.Handle("/api-ui", ember.ServeAsset())
	router.Handle("/assets/rancher-ui-driver-linode", emberAlwaysOffline.ServeAsset())
	router.Handle("/assets", ember.ServeAsset())
	router.Handle("/dashboard/", vue.IndexFileOnNotFound())
	router.Handle("/ember-fetch", ember.ServeAsset())
	router.Handle("/engines-dist", ember.ServeAsset())
	router.Handle("/static", ember.ServeAsset())
	router.Handle("/translations", ember.ServeAsset())
	router.NotFound(emberIndexUnlessAPI().ServeHTTP)

	return router
}

func emberIndexUnlessAPI() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if parse.IsBrowser(req, true) {
			emberIndex.ServeHTTP(rw, req)
		} else {
			http.NotFound(rw, req)
		}
	})
}

func PreferredIndex() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, "/dashboard/", http.StatusFound)
	})
}
