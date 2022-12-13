package projects

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/pkg/kv"
)

type stripNS struct {
	writer types.ResponseWriter
}

func (s stripNS) Write(apiOp *types.APIRequest, code int, obj types.APIObject) {
	if chi.URLParam(apiOp.Request, "clusterID") == "" {
		prefix := apiOp.Namespace + "/"
		if strings.HasPrefix(obj.ID, prefix) {
			_, obj.ID = kv.RSplit(obj.ID, "/")
		}
	}
	s.writer.Write(apiOp, code, obj)
}

func (s stripNS) WriteList(apiOp *types.APIRequest, code int, obj types.APIObjectList) {
	if chi.URLParam(apiOp.Request, "clusterID") == "" {
		prefix := apiOp.Namespace + "/"
		for i := range obj.Objects {
			if strings.HasPrefix(obj.Objects[i].ID, prefix) {
				_, obj.Objects[i].ID = kv.RSplit(obj.Objects[i].ID, "/")
			}
		}
	}
	s.writer.WriteList(apiOp, code, obj)
}
