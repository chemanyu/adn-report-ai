package report

import (
	"strconv"
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/errcode"
	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

func (l *ReportLogic) List(in types.ListUploadsRequest, u types.User) (*types.UploadListResponse, error) {
	filter := model.UploadFilter{OwnerID: ownerScope(u), Query: strings.TrimSpace(in.Query), Page: pageNumber(in.Page)}
	if in.AccountID != "" {
		id, err := strconv.ParseInt(in.AccountID, 10, 64)
		if err != nil || id < 1 {
			return nil, errcode.New(400, "账户筛选无效")
		}
		filter.AccountID = id
	}
	list, err := l.svcCtx.UploadModel.List(l.ctx, filter)
	if err != nil {
		return nil, err
	}
	result := &types.UploadListResponse{Items: make([]types.Upload, 0, len(list.Items)), Total: list.Total, Page: filter.Page, PageSize: 20, RowCount: list.RowCount, TotalAmount: list.TotalAmount, AccountCount: list.AccountCount}
	for _, item := range list.Items {
		result.Items = append(result.Items, types.Upload(item))
	}
	return result, nil
}
