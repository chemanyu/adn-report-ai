package types

type User struct {
	ID       int64  `json:"id"`
	Identity string `json:"identity"`
	Name     string `json:"name"`
	Admin    bool   `json:"admin"`
}
type LocalLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type AuthConfigResponse struct {
	DingTalk   bool `json:"dingtalk"`
	LocalAdmin bool `json:"local_admin"`
}
type OKResponse struct {
	OK bool `json:"ok"`
}
