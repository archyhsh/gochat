// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package relation

import (
	"net/http"

	"github.com/archyhsh/gochat/api/internal/logic/relation"
	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func HandleApplyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HandleApplyRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := relation.NewHandleApplyLogic(r.Context(), svcCtx)
		resp, err := l.HandleApply(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
