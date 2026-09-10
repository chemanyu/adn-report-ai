package report

import (
	"context"

	"github.com/chemanyu/adn-report-ai/internal/svc"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

type ReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReportLogic {
	return &ReportLogic{ctx: ctx, svcCtx: svcCtx}
}

// ownerScope derives access from the authenticated user, never from operator names.
func ownerScope(u types.User) *int64 {
	if u.Admin {
		return nil
	}
	return &u.ID
}
func pageNumber(page int) int {
	if page < 1 || page > 1000000 {
		return 1
	}
	return page
}
