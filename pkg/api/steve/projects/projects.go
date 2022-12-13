package projects

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/types"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	"github.com/rancher/steve/pkg/accesscontrol"
	"github.com/rancher/steve/pkg/attributes"
	"github.com/rancher/steve/pkg/auth"
	"github.com/rancher/steve/pkg/client"
	"github.com/rancher/steve/pkg/schema"
	steveserver "github.com/rancher/steve/pkg/server"
	"github.com/rancher/steve/pkg/stores/proxy"
)

type projectServer struct {
	ctx          context.Context
	asl          accesscontrol.AccessSetLookup
	auth         auth.Middleware
	cf           *client.Factory
	clusterLinks []string
}

func Projects(ctx context.Context, server *steveserver.Server) (func(http.Handler) http.Handler, error) {
	s := projectServer{}
	if err := s.Setup(ctx, server); err != nil {
		return nil, err
	}
	return s.middleware(), nil
}

func (s *projectServer) Setup(ctx context.Context, server *steveserver.Server) error {
	s.ctx = ctx
	s.asl = server.AccessSetLookup
	s.cf = server.ClientFactory

	server.SchemaFactory.AddTemplate(schema.Template{
		ID: "management.cattle.io.cluster",
		Formatter: func(request *types.APIRequest, resource *types.RawResource) {
			for _, link := range s.clusterLinks {
				resource.Links[link] = request.URLBuilder.Link(resource.Schema, resource.ID, link)
			}
		},
	})

	return nil
}

func (s *projectServer) newSchemas() *types.APISchemas {
	store := proxy.NewProxyStore(s.cf, nil, s.asl)
	schemas := types.EmptyAPISchemas()

	schemas.MustImportAndCustomize(v3.Project{}, func(schema *types.APISchema) {
		schema.Store = store
		attributes.SetNamespaced(schema, true)
		attributes.SetGroup(schema, v3.GroupName)
		attributes.SetVersion(schema, "v3")
		attributes.SetKind(schema, "Project")
		attributes.SetResource(schema, "projects")
		attributes.SetVerbs(schema, []string{"create", "list", "get", "delete", "update", "watch", "patch"})
		s.clusterLinks = append(s.clusterLinks, "projects")
	})

	return schemas
}

func (s *projectServer) newAPIHandler() http.Handler {
	server := server.DefaultAPIServer()
	for k, v := range server.ResponseWriters {
		server.ResponseWriters[k] = stripNS{writer: v}
	}

	s.clusterLinks = []string{
		"subscribe",
		"schemas",
	}

	sf := schema.NewCollection(s.ctx, server.Schemas, s.asl)
	sf.Reset(s.newSchemas().Schemas)

	return schema.WrapServer(sf, server)
}

func (s *projectServer) middleware() func(http.Handler) http.Handler {
	server := s.newAPIHandler()
	server = prefix(server)

	router := chi.NewRouter()
	//router.Use(middleware.RequestID)
	//router.Use(middleware.RealIP)
	//router.Use(middleware.Logger)
	//router.Use(middleware.Recoverer)
	//TODO: MUX
	//router.UseEncodedPath()
	//TODO: MUX
	//router.Path("/v1/management.cattle.io.clusters/{namespace}").Queries("link", "{type:projects?}").Handler(server)
	router.Handle("/v1/management.cattle.io.clusters/{namespace}?link={link}&type=projects", server)
	router.Handle("/v1/management.cattle.io.clusters/{namespace}/{type}", server)
	router.Handle("/v1/management.cattle.io.clusters/{namespace}/{type}/{name}", server)
	router.Handle("/v1/management.cattle.io.clusters/{clusterID}/{type}/{namespace}/{name}", server)

	return func(next http.Handler) http.Handler {
		router.NotFound(next.ServeHTTP)
		return router
	}
}

func prefix(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		clusterID := chi.URLParam(req, "clusterID")
		var prefix string
		if clusterID != "" {
			prefix = "/v1/management.cattle.io.clusters/" + clusterID
		} else {
			namespace := chi.URLParam(req, "namespace")
			prefix = "/v1/management.cattle.io.clusters/" + namespace
		}

		ctx := context.WithValue(req.Context(), "prefix", prefix)

		next.ServeHTTP(rw, req.WithContext(ctx))
	})
}
