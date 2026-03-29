// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package relation

import (
	"net/http"

	"github.com/archyhsh/gochat/api/internal/logic/relation"
	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetApplyListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := relation.NewGetApplyListLogic(r.Context(), svcCtx)
		resp, err := l.GetApplyList()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
