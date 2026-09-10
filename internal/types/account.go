package types

type Account struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}
type CreateAccountRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}
