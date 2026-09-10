package account

import (
	"net/http"

	accountlogic "github.com/chemanyu/adn-report-ai/internal/logic/account"
	"github.com/chemanyu/adn-report-ai/internal/middleware"
	"github.com/chemanyu/adn-report-ai/internal/response"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

func ListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := accountlogic.NewAccountLogic(r.Context(), svcCtx).List()
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, 200, result)
	}
}
func CreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in types.CreateAccountRequest
		if !response.Decode(w, r, &in) {
			return
		}
		result, err := accountlogic.NewAccountLogic(r.Context(), svcCtx).Create(in, middleware.CurrentUser(r.Context()))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, 201, result)
	}
}
