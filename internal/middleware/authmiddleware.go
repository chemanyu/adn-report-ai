package middleware

import (
	"context"
	"net/http"

	authlogic "github.com/chemanyu/adn-report-ai/internal/logic/auth"
	"github.com/chemanyu/adn-report-ai/internal/response"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

type userContextKey struct{}

func CurrentUser(ctx context.Context) types.User {
	user, _ := ctx.Value(userContextKey{}).(types.User)
	return user
}
func Auth(svcCtx *svc.ServiceContext) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, err := authlogic.NewAuthLogic(r.Context(), svcCtx).Current(response.CookieValue(r, "adn_session"))
			if err != nil {
				response.Error(w, err)
				return
			}
			next(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
		}
	}
}
