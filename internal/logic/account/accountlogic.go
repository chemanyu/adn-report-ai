package account

import (
	"context"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

type AccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AccountLogic {
	return &AccountLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *AccountLogic) List() ([]types.Account, error) {
	accounts, err := l.svcCtx.AccountModel.List(l.ctx)
	if err != nil {
		return nil, err
	}
	result := make([]types.Account, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, types.Account(a))
	}
	return result, nil
}
func (l *AccountLogic) Create(in types.CreateAccountRequest, u types.User) (types.Account, error) {
	name, code := strings.TrimSpace(in.Name), strings.TrimSpace(in.Code)
	if name == "" || code == "" || len([]rune(name)) > 191 || len([]rune(code)) > 191 {
		return types.Account{}, errcode.New(400, "请填写账户名称和 ADN 账户标识（最多 191 字）")
	}
	a, err := l.svcCtx.AccountModel.Insert(l.ctx, name, code, u.ID)
	if model.IsDuplicate(err) {
		return types.Account{}, errcode.New(409, "该 ADN 账户标识已存在，请直接选择")
	}
	return types.Account(a), err
}
