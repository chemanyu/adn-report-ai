package auth

import (
	"errors"
	"net/http"
	"strings"

	authlogic "github.com/chemanyu/adn-report-ai/internal/logic/auth"
	"github.com/chemanyu/adn-report-ai/internal/middleware"
	"github.com/chemanyu/adn-report-ai/internal/response"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

func setCookie(w http.ResponseWriter, svcCtx *svc.ServiceContext, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: svcCtx.Config.BasePath() + "/", HttpOnly: true, Secure: strings.HasPrefix(svcCtx.Config.BaseURL, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func ConfigHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, 200, authlogic.NewAuthLogic(r.Context(), svcCtx).Config())
	}
}
func MeHandler(_ *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, 200, middleware.CurrentUser(r.Context()))
	}
}
func LocalLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := authlogic.NewAuthLogic(r.Context(), svcCtx)
		// Count malformed requests too, before decoding a possible credential body.
		if err := l.CheckLoginLimit(); err != nil {
			response.Error(w, err)
			return
		}
		var in types.LocalLoginRequest
		if !response.Decode(w, r, &in) {
			return
		}
		token, err := l.LocalLogin(in, response.CookieValue(r, "adn_session"))
		if err != nil {
			response.Error(w, err)
			return
		}
		setCookie(w, svcCtx, "adn_session", token, authlogic.SessionAge)
		response.JSON(w, 200, types.OKResponse{OK: true})
	}
}
func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, authorizeURL, err := authlogic.NewAuthLogic(r.Context(), svcCtx).StartOAuth()
		if err != nil {
			response.Error(w, err)
			return
		}
		setCookie(w, svcCtx, "adn_oauth", state, 600)
		http.Redirect(w, r, authorizeURL, http.StatusFound)
	}
}
func CallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := q.Get("authCode")
		if code == "" {
			code = q.Get("code")
		}
		token, clearState, err := authlogic.NewAuthLogic(r.Context(), svcCtx).Callback(q.Get("state"), response.CookieValue(r, "adn_oauth"), code, response.CookieValue(r, "adn_session"))
		if clearState {
			setCookie(w, svcCtx, "adn_oauth", "", -1)
		}
		if errors.Is(err, authlogic.ErrOAuthState) {
			http.Redirect(w, r, svcCtx.Config.BasePath()+"/?error=oauth_state", 303)
			return
		}
		if errors.Is(err, authlogic.ErrOAuthExchange) {
			http.Redirect(w, r, svcCtx.Config.BasePath()+"/?error=oauth_failed", 303)
			return
		}
		if err != nil {
			response.Error(w, err)
			return
		}
		setCookie(w, svcCtx, "adn_session", token, authlogic.SessionAge)
		http.Redirect(w, r, svcCtx.Config.BasePath()+"/", 303)
	}
}
func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authlogic.NewAuthLogic(r.Context(), svcCtx).Logout(response.CookieValue(r, "adn_session")); err != nil {
			response.Error(w, err)
			return
		}
		setCookie(w, svcCtx, "adn_session", "", -1)
		response.JSON(w, 200, types.OKResponse{OK: true})
	}
}
