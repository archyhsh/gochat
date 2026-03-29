// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"net/http"

	"github.com/archyhsh/gochat/api/internal/logic/group"
	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetGroupRequestsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := group.NewGetGroupRequestsLogic(r.Context(), svcCtx)
		resp, err := l.GetGroupRequests()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
