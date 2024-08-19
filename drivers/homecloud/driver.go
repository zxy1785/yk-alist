package homecloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/cron"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type HomeCloud struct {
	model.Storage
	Addition
	cron    *cron.Cron
	Account string
}

func (d *HomeCloud) Config() driver.Config {
	return config
}

func (d *HomeCloud) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *HomeCloud) Init(ctx context.Context) error {
	if d.Authorization == "" {
		return fmt.Errorf("authorization is empty")
	}
	if d.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}
	if d.GroupID == "" {
		return fmt.Errorf("GroupID is empty")
	}

	d.RootFolderID = "/"
	return nil
}

func (d *HomeCloud) Drop(ctx context.Context) error {
	if d.cron != nil {
		d.cron.Stop()
	}
	return nil
}

func (d *HomeCloud) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	return d.familyGetFiles(dir.GetID())
}

func (d *HomeCloud) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var url string
	var err error
	url, err = d.getLink(file.GetID())
	if err != nil {
		return nil, err
	}
	return &model.Link{URL: url}, nil
}

func (d *HomeCloud) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	var err error
	data := base.Json{
		"parentDirId": parentDir.GetID(),
		"dirName":     dirName,
		"category":    0,
		"userId":      d.UserID,
		"groupId":     d.GroupID,
	}
	pathname := "/storage/addDirectory/v1"
	_, err = d.post(pathname, data, nil)
	return err
}

func (d *HomeCloud) Move(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	data := base.Json{
		"fileIds":     []string{srcObj.GetID()},
		"targetDirId": dstDir.GetID(),
		"userId":      d.UserID,
		"groupId":     d.GroupID,
	}
	pathname := "/storage/batchMoveFile/v1"
	_, err := d.post(pathname, data, nil)
	if err != nil {
		return nil, err
	}
	return srcObj, nil
}

func (d *HomeCloud) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	var err error
	data := base.Json{
		"fileId":   srcObj.GetID(),
		"fileName": newName,
		"userId":   d.UserID,
		"groupId":  d.GroupID,
	}
	pathname := "/storage/updateFileName/v1"
	_, err = d.post(pathname, data, nil)

	return err
}

func (d *HomeCloud) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	data := base.Json{
		"fileIds":       []string{srcObj.GetID()},
		"targetDirId":   dstDir.GetID(),
		"targetGroupId": d.GroupID,
		"userId":        d.UserID,
		"groupId":       d.GroupID,
	}
	pathname := "/storage/batchCopyFile/v1"
	_, err := d.post(pathname, data, nil)
	return err
}

func (d *HomeCloud) Remove(ctx context.Context, obj model.Obj) error {
	data := base.Json{
		"fileIds": []string{obj.GetID()},
		"userId":  d.UserID,
		"groupId": d.GroupID,
	}
	pathname := "/storage/batchDeleteFile/v1"
	_, err := d.post(pathname, data, nil)
	return err
}

const (
	_  = iota //ignore first value by assigning to blank identifier
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

func getPartSize(size int64) int64 {
	// 网盘对于分片数量存在上限
	if size/GB > 30 {
		return 512 * MB
	}
	return 100 * MB
}

func (d *HomeCloud) Put(ctx context.Context, dstDir model.Obj, stream model.FileStreamer, up driver.UpdateProgress) error {
	switch d.Addition.Type {
	case MetaPersonalNew:
		var err error
		fullHash := stream.GetHash().GetHash(utils.SHA256)
		if len(fullHash) <= 0 {
			tmpF, err := stream.CacheFullInTempFile()
			if err != nil {
				return err
			}
			fullHash, err = utils.HashFile(utils.SHA256, tmpF)
			if err != nil {
				return err
			}
		}
		// return errs.NotImplement
		data := base.Json{
			"contentHash":          fullHash,
			"contentHashAlgorithm": "SHA256",
			"contentType":          "application/octet-stream",
			"parallelUpload":       false,
			"partInfos": []base.Json{{
				"parallelHashCtx": base.Json{
					"partOffset": 0,
				},
				"partNumber": 1,
				"partSize":   stream.GetSize(),
			}},
			"size":           stream.GetSize(),
			"parentFileId":   dstDir.GetID(),
			"name":           stream.GetName(),
			"type":           "file",
			"fileRenameMode": "auto_rename",
		}
		pathname := "/hcy/file/create"
		var resp PersonalUploadResp
		_, err = d.personalPost(pathname, data, &resp)
		if err != nil {
			return err
		}

		if resp.Data.Exist || resp.Data.RapidUpload {
			return nil
		}

		// Progress
		p := driver.NewProgress(stream.GetSize(), up)

		// Update Progress
		r := io.TeeReader(stream, p)

		req, err := http.NewRequest("PUT", resp.Data.PartInfos[0].UploadUrl, r)
		if err != nil {
			return err
		}
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Length", fmt.Sprint(stream.GetSize()))
		req.Header.Set("Origin", "https://yun.139.com")
		req.Header.Set("Referer", "https://yun.139.com/")
		req.ContentLength = stream.GetSize()

		res, err := base.HttpClient.Do(req)
		if err != nil {
			return err
		}

		_ = res.Body.Close()
		log.Debugf("%+v", res)
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}

		data = base.Json{
			"contentHash":          fullHash,
			"contentHashAlgorithm": "SHA256",
			"fileId":               resp.Data.FileId,
			"uploadId":             resp.Data.UploadId,
		}
		_, err = d.personalPost("/hcy/file/complete", data, nil)
		if err != nil {
			return err
		}
		return nil
	case MetaPersonal:
		fallthrough
	case MetaFamily:
		data := base.Json{
			"manualRename": 2,
			"operation":    0,
			"fileCount":    1,
			"totalSize":    0, // 去除上传大小限制
			"uploadContentList": []base.Json{{
				"contentName": stream.GetName(),
				"contentSize": 0, // 去除上传大小限制
				// "digest": "5a3231986ce7a6b46e408612d385bafa"
			}},
			"parentCatalogID": dstDir.GetID(),
			"newCatalogName":  "",
			"commonAccountInfo": base.Json{
				"account":     d.Account,
				"accountType": 1,
			},
		}
		pathname := "/orchestration/personalCloud/uploadAndDownload/v1.0/pcUploadFileRequest"
		if d.isFamily() {
			cataID := dstDir.GetID()
			path := cataID
			seqNo, _ := uuid.NewUUID()
			data = base.Json{
				"cloudID":      d.CloudID,
				"path":         path,
				"operation":    0,
				"cloudType":    1,
				"catalogType":  3,
				"manualRename": 2,
				"fileCount":    1,
				"totalSize":    0,
				"uploadContentList": []base.Json{{
					"contentName": stream.GetName(),
					"contentSize": 0,
				}},
				"seqNo": seqNo,
				"commonAccountInfo": base.Json{
					"account":     d.Account,
					"accountType": 1,
				},
			}
			pathname = "/orchestration/familyCloud-rebuild/content/v1.0/getFileUploadURL"
			//return errs.NotImplement
		}
		var resp UploadResp
		_, err := d.post(pathname, data, &resp)
		if err != nil {
			return err
		}
		// Progress
		p := driver.NewProgress(stream.GetSize(), up)

		var partSize = getPartSize(stream.GetSize())
		part := (stream.GetSize() + partSize - 1) / partSize
		if part == 0 {
			part = 1
		}
		for i := int64(0); i < part; i++ {
			if utils.IsCanceled(ctx) {
				return ctx.Err()
			}

			start := i * partSize
			byteSize := stream.GetSize() - start
			if byteSize > partSize {
				byteSize = partSize
			}

			limitReader := io.LimitReader(stream, byteSize)
			// Update Progress
			r := io.TeeReader(limitReader, p)
			req, err := http.NewRequest("POST", resp.Data.UploadResult.RedirectionURL, r)
			if err != nil {
				return err
			}

			req = req.WithContext(ctx)
			req.Header.Set("Content-Type", "text/plain;name="+unicode(stream.GetName()))
			req.Header.Set("contentSize", strconv.FormatInt(stream.GetSize(), 10))
			req.Header.Set("range", fmt.Sprintf("bytes=%d-%d", start, start+byteSize-1))
			req.Header.Set("uploadtaskID", resp.Data.UploadResult.UploadTaskID)
			req.Header.Set("rangeType", "0")
			req.ContentLength = byteSize

			res, err := base.HttpClient.Do(req)
			if err != nil {
				return err
			}
			_ = res.Body.Close()
			log.Debugf("%+v", res)
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d", res.StatusCode)
			}
		}

		return nil
	default:
		return errs.NotImplement
	}
}

func (d *HomeCloud) Other(ctx context.Context, args model.OtherArgs) (interface{}, error) {
	switch d.Addition.Type {
	case MetaPersonalNew:
		var resp base.Json
		var uri string
		data := base.Json{
			"category": "video",
			"fileId":   args.Obj.GetID(),
		}
		switch args.Method {
		case "video_preview":
			uri = "/hcy/videoPreview/getPreviewInfo"
		default:
			return nil, errs.NotSupport
		}
		_, err := d.personalPost(uri, data, &resp)
		if err != nil {
			return nil, err
		}
		return resp["data"], nil
	default:
		return nil, errs.NotImplement
	}
}

var _ driver.Driver = (*HomeCloud)(nil)
