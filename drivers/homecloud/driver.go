package homecloud

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/cron"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/pkg/utils/random"
)

type HomeCloud struct {
	model.Storage
	Addition
	AccessToken string
	UserID      string
	cron        *cron.Cron
	Account     string
}

func (d *HomeCloud) Config() driver.Config {
	return config
}

func (d *HomeCloud) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *HomeCloud) Init(ctx context.Context) error {
	if d.RefreshToken == "" {
		return fmt.Errorf("RefreshToken is empty")
	}

	if len(d.Addition.RootFolderID) == 0 {
		d.RootFolderID = "/"
	}

	err := d.refreshToken()
	if err != nil {
		return err
	}

	d.cron = cron.NewCron(time.Hour * 10)
	d.cron.Do(func() {
		err := d.refreshToken()
		if err != nil {
			return
		}
	})

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

	link := &model.Link{
		URL: url,
	}

	// 创建Header 否则新上传文件无法使用
	header := make(http.Header)
	header.Add("Cookie", "H_TOKEN="+d.AccessToken)
	link.Header = header

	return link, nil
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
	// 复制会占用空间 所以先屏蔽代码
	// data := base.Json{
	// 	"fileIds":       []string{srcObj.GetID()},
	// 	"targetDirId":   dstDir.GetID(),
	// 	"targetGroupId": d.GroupID,
	// 	"userId":        d.UserID,
	// 	"groupId":       d.GroupID,
	// }
	// pathname := "/storage/batchCopyFile/v1"
	// _, err := d.post(pathname, data, nil)
	// return err
	return errs.NotImplement
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
	var err error

	h := md5.New()
	// need to calculate md5 of the full content
	tempFile, err := stream.CacheFullInTempFile()
	if err != nil {
		return err
	}
	defer func() {
		_ = tempFile.Close()
	}()
	if _, err = io.Copy(h, tempFile); err != nil {
		return err
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	etag := hex.EncodeToString(h.Sum(nil))

	// return errs.NotImplement
	data := base.Json{
		"userId":       d.UserID,
		"groupId":      d.GroupID,
		"dirId":        dstDir.GetID(),
		"fileName":     stream.GetName(),
		"fileMd5":      etag,
		"fileSize":     stream.GetSize(),
		"fileCategory": 99,
	}

	pathname := "/storage/addFileUploadTask/v1"
	var resp PersonalUploadResp
	_, err = d.post(pathname, data, &resp)
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
		// Update Progress
		//r := io.TeeReader(limitReader, p)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		filePart, err := writer.CreateFormFile("partFile", stream.GetName())
		if err != nil {
			return err
		}
		_, err = io.Copy(filePart, r)
		if err != nil {
			return err
		}

		isDone := false

		if i == (part - 1) {
			isDone = true
		}

		_ = writer.WriteField("uploadId", resp.Data.UploadId)
		_ = writer.WriteField("isComplete", strconv.FormatBool(isDone))
		_ = writer.WriteField("rangeStart", strconv.Itoa(int(start)))

		err = writer.Close()
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", resp.Data.UploadUrl, body)
		if err != nil {
			return err
		}
		requestID := random.String(12)
		pbody, err := utils.Json.Marshal(body)

		if err != nil {
			return err
		}

		timestamp := fmt.Sprintf("%.3f", float64(time.Now().UnixNano())/1e6)
		h := sha1.New()
		var sha1Hash string

		if pbody == nil {
			h.Write([]byte("{}"))
			sha1Hash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		} else {
			h.Write(pbody)
			sha1Hash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		}

		uppathname := "/upload/upload/uploadFilePart/v1"
		encStr := fmt.Sprintf("%s;%s;%s;Bearer %s;%s", uppathname, sha1Hash, requestID, d.AccessToken, timestamp)
		signature := strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(encStr))))

		req = req.WithContext(ctx)
		req.Header.Add("Accept", "*/*")
		req.Header.Add("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Add("Authorization", "Bearer "+d.AccessToken)
		req.Header.Add("Origin", "https://homecloud.komect.com")
		req.Header.Add("Referer", "https://homecloud.komect.com/disk/main/familyspace")
		req.Header.Add("Request-Id", requestID)
		req.Header.Add("Signature", signature)
		req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
		req.Header.Add("X-Requested-With", "XMLHttpRequest")
		req.Header.Add("X-User-Agent", "Web|Chrome 127.0.0.0||OS X|homecloudWebDisk_1.1.1||yunpan 1.1.1|unknown")
		req.Header.Add("sec-ch-ua", "\"Not)A;Brand\";v=\"99\", \"Google Chrome\";v=\"127\", \"Chromium\";v=\"127\"")
		req.Header.Add("sec-ch-ua-mobile", "?0")
		req.Header.Add("sec-ch-ua-platform", "\"macOS\"")
		req.Header.Add("userId", d.UserID)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := base.HttpClient.Do(req)
		if err != nil {
			return err
		}
		_ = res.Body.Close()
		//log.Debugf("%+v", res)
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}

	}

	// url, err := d.getLink(resp.Data.FileId)
	// if err != nil {
	// 	return fmt.Errorf("can not get file donwnload url")
	// }

	// _, err = base.RestyClient.R().
	// 	SetHeader("Cookie", "H_TOKEN="+d.AccessToken).
	// 	SetHeader("Range", "bytes=0-100").
	// 	Get(url)
	// if err != nil {
	// 	return fmt.Errorf("can not active file")
	// }

	return nil
}

func (d *HomeCloud) Other(ctx context.Context, args model.OtherArgs) (interface{}, error) {
	return nil, errs.NotImplement
}

var _ driver.Driver = (*HomeCloud)(nil)
