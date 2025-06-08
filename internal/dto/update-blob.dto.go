package dto

import (
	"liangyuanguo/aw/blob/pkg/model"
)

type UpdateBlobReq struct {
	ID string `json:"id"`

	IsPublic bool   `json:"isPublic"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`

	ContentType string `json:"contentType"`
	Categories  string `json:"categories"`

	Tags string `json:"tags"`
}

//func NewUpdateBlobReq(ctx *gin.Context) (*UpdateBlobReq, error) {
//	fileName := ctx.Param("id")
//	if fileName == "" {
//		return nil, fmt.Errorf("file name is required")
//	}
//
//	files := strings.Split(fileName, ".")
//	fileID := fileName
//	if len(files) > 1 {
//		fileID = files[0]
//	}
//
//	contentType := ctx.GetHeader("Content-Type")
//	if contentType == "" {
//		if len(files) > 1 {
//			contentType = mime.TypeByExtension(files[1])
//		} else {
//			contentType = "application/octet-stream"
//		}
//	}
//
//	return &UpdateBlobReq{
//		Name: fileName,
//	}, nil
//}

func (req *UpdateBlobReq) Validate() error {
	return nil

}

func (req *UpdateBlobReq) Apply(b *model.Blob) {
	b.Name = req.Name
	b.Desc = req.Desc
	b.ContentType = req.ContentType
	b.Categories = req.Categories
	b.Tags = req.Tags
	b.IsPublic = req.IsPublic
}
