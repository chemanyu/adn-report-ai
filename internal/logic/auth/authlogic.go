package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/security"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

var ErrOAuthState = errors.New("OAuth state 无效或已过期")
var ErrOAuthExchange = errors.New("钉钉授权失败")

const SessionAge = 7 * 86400

type AuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AuthLogic {
	return &AuthLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *AuthLogic) Config() types.AuthConfigResponse {
	a := l.svcCtx.Config.Auth
	return types.AuthConfigResponse{DingTalk: a.ClientID != "" && a.ClientSecret != "", LocalAdmin: a.AdminPassword != ""}
}
func (l *AuthLogic) isAdmin(identity string) bool {
	a := l.svcCtx.Config.Auth
	if identity == "local:"+a.AdminUsername {
		return a.AdminPassword != ""
	}
	for _, id := range a.AdminUnionIDs {
		if identity == "ding:"+strings.TrimSpace(id) {
			return true
		}
	}
	return false
}
func (l *AuthLogic) Current(token string) (types.User, error) {
	if token == "" {
		return types.User{}, errcode.New(401, "请先登录")
	}
	u, err := l.svcCtx.AuthModel.FindSession(l.ctx, security.Hash(token))
	if errors.Is(err, model.ErrNotFound) {
		return types.User{}, errcode.New(401, "请先登录")
	}
	if err != nil {
		return types.User{}, err
	}
	return types.User{ID: u.ID, Identity: u.Identity, Name: u.Name, Admin: l.isAdmin(u.Identity)}, nil
}
func (l *AuthLogic) SignIn(identity, name, oldToken string) (string, error) {
	if name == "" {
		name = "钉钉用户"
	}
	if len([]rune(name)) > 191 || len(identity) > 191 {
		return "", fmt.Errorf("用户信息过长")
	}
	oldHash := ""
	if oldToken != "" {
		oldHash = security.Hash(oldToken)
	}
	token := security.RandomToken()
	err := l.svcCtx.AuthModel.CreateSession(l.ctx, identity, name, oldHash, security.Hash(token), time.Now().UTC().Add(SessionAge*time.Second))
	if err != nil {
		return "", err
	}
	return token, nil
}
func (l *AuthLogic) CheckLoginLimit() error {
	if !l.svcCtx.LoginLimiter.Allow() {
		return errcode.New(429, "登录尝试过多，请一分钟后重试")
	}
	return nil
}
func (l *AuthLogic) LocalLogin(in types.LocalLoginRequest, oldToken string) (string, error) {
	a := sha256.Sum256([]byte(in.Password))
	b := sha256.Sum256([]byte(l.svcCtx.Config.Auth.AdminPassword))
	cfg := l.svcCtx.Config.Auth
	if cfg.AdminPassword == "" || in.Username != cfg.AdminUsername || subtle.ConstantTimeCompare(a[:], b[:]) != 1 {
		return "", errcode.New(401, "账号或密码错误")
	}
	return l.SignIn("local:"+cfg.AdminUsername, "管理员", oldToken)
}
func (l *AuthLogic) StartOAuth() (state, authorizeURL string, err error) {
	cfg := l.svcCtx.Config
	if !l.Config().DingTalk {
		return "", "", errcode.New(503, "尚未配置钉钉应用，请设置 ClientID 和 ClientSecret")
	}
	state = security.RandomToken()
	if err = l.svcCtx.AuthModel.CreateOAuthState(l.ctx, security.Hash(state), time.Now().UTC().Add(10*time.Minute)); err != nil {
		return "", "", err
	}
	q := url.Values{"client_id": {cfg.Auth.ClientID}, "redirect_uri": {cfg.BaseURL + "/auth/callback"}, "response_type": {"code"}, "scope": {"openid"}, "prompt": {"consent"}, "state": {state}}
	return state, "https://login.dingtalk.com/oauth2/auth?" + q.Encode(), nil
}

// Callback returns clearState only after the callback matches the browser binding.
func (l *AuthLogic) Callback(state, browserState, code, oldToken string) (token string, clearState bool, err error) {
	if state == "" || browserState == "" || subtle.ConstantTimeCompare([]byte(state), []byte(browserState)) != 1 {
		return "", false, ErrOAuthState
	}
	used, err := l.svcCtx.AuthModel.ConsumeOAuthState(l.ctx, security.Hash(state))
	if err != nil {
		return "", true, err
	}
	if !used {
		return "", true, ErrOAuthState
	}
	if code == "" {
		return "", true, ErrOAuthExchange
	}
	cfg := l.svcCtx.Config.Auth
	profile, err := l.svcCtx.DingTalk.Exchange(l.ctx, cfg.ClientID, cfg.ClientSecret, code)
	if err != nil {
		return "", true, ErrOAuthExchange
	}
	token, err = l.SignIn("ding:"+profile.UnionID, profile.Nick, oldToken)
	return token, true, err
}
func (l *AuthLogic) Logout(token string) error {
	if token == "" {
		return nil
	}
	return l.svcCtx.AuthModel.DeleteSession(l.ctx, security.Hash(token))
}
