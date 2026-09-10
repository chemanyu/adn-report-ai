package report

import (
	"strings"

	"github.com/chemanyu/adn-report-ai/internal/model"
	"github.com/chemanyu/adn-report-ai/internal/types"
)

func (l *ReportLogic) List(in types.ListUploadsRequest, u types.User) (*types.UploadListResponse, error) {
	filter := model.UploadFilter{OwnerID: ownerScope(u), Query: strings.TrimSpace(in.Query), Page: pageNumber(in.Page)}
	list, err := l.svcCtx.UploadModel.List(l.ctx, filter)
	if err != nil {
		return nil, err
	}
	result := &types.UploadListResponse{Items: make([]types.Upload, 0, len(list.Items)), Total: list.Total, Page: filter.Page, PageSize: 20, RowCount: list.RowCount, TotalAmount: list.TotalAmount, AdvertiserCount: list.AdvertiserCount}
	for _, item := range list.Items {
		result.Items = append(result.Items, types.Upload(item))
	}
	return result, nil
}
