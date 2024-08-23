package homecloud

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// do others that not defined in Driver interface
func (d *HomeCloud) isFamily() bool {
	return false
}

func encodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	r = strings.Replace(r, "%21", "!", -1)
	r = strings.Replace(r, "%27", "'", -1)
	r = strings.Replace(r, "%28", "(", -1)
	r = strings.Replace(r, "%29", ")", -1)
	r = strings.Replace(r, "%2A", "*", -1)
	return r
}

func calSign(body, ts, randStr string) string {
	body = encodeURIComponent(body)
	strs := strings.Split(body, "")
	sort.Strings(strs)
	body = strings.Join(strs, "")
	body = base64.StdEncoding.EncodeToString([]byte(body))
	res := utils.GetMD5EncodeStr(body) + utils.GetMD5EncodeStr(ts+":"+randStr)
	res = strings.ToUpper(utils.GetMD5EncodeStr(res))
	return res
}

func getTime(t string) time.Time {
	stamp, _ := time.ParseInLocation("20060102150405", t, utils.CNLoc)
	return stamp
}

func (d *HomeCloud) refreshToken() error {
	pathname := "/auth/refreshToken/v2"
	url := "https://homecloud.komect.com/front" + pathname

	data := base.Json{
		"refresh_token": d.RefreshToken,
		"scope":         "sdk",
	}

	requestID := random.String(12)
	body, err := utils.Json.Marshal(data)

	if err != nil {
		return err
	}

	timestamp := fmt.Sprintf("%.3f", float64(time.Now().UnixNano())/1e6)
	h := sha1.New()
	var sha1Hash string

	if body == nil {
		h.Write([]byte("{}"))
		sha1Hash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	} else {
		h.Write(body)
		sha1Hash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	}

	encStr := fmt.Sprintf("%s;%s;%s;Basic VTdUOU1xSHpVbklqeWdETzppQzZCU25QaExyODZGZmJX;%s", pathname, sha1Hash, requestID, timestamp)
	signature := strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(encStr))))

	var resp RefreshTokenResp
	var e FrontResp
	req := base.RestyClient.R()
	req.SetHeaders(map[string]string{
		"Accept":        "application/json, text/plain, */*",
		"Authorization": "Basic VTdUOU1xSHpVbklqeWdETzppQzZCU25QaExyODZGZmJX",
		"Content-Type":  "application/json",
		"X-User-Agent":  "Web|Chrome 127.0.0.0||OS X|homecloudWebDisk_1.1.1||yunpan 1.1.1|unknown",
		"Timestamp":     timestamp,
		"Signature":     signature,
		"Request-Id":    requestID,
		"userId":        "",
	})
	req.SetBody(data)
	req.SetResult(&resp)
	req.SetError(&e)
	_, err = req.Post(url)
	//fmt.Println(string(res.Body()))

	if err != nil {
		return err
	}
	// if e.Ret != 200 {
	// 	return fmt.Errorf("failed to refresh token: %s", e.Reason)
	// }
	if resp.Data.RefreshToken == "" {
		return errors.New("failed to refresh token: refresh token is empty")
	}
	d.RefreshToken, d.AccessToken, d.UserID = resp.Data.RefreshToken, resp.Data.AccessToken, resp.Data.UserID
	op.MustSaveDriverStorage(d)
	return nil
}

func (d *HomeCloud) request(pathname string, method string, callback base.ReqCallback, resp interface{}) ([]byte, error) {
	url := "https://homecloud.komect.com/front" + pathname
	req := base.RestyClient.R()
	requestID := random.String(12)
	//ts := time.Now().Format("2006-01-02 15:04:05")
	if callback != nil {
		callback(req)
	}
	body, err := utils.Json.Marshal(req.Body)

	if err != nil {
		return nil, err
	}

	timestamp := fmt.Sprintf("%.3f", float64(time.Now().UnixNano())/1e6)
	h := sha1.New()
	var sha1Hash string

	if body == nil {
		h.Write([]byte("{}"))
		sha1Hash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	} else {
		h.Write(body)
		sha1Hash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	}

	encStr := fmt.Sprintf("%s;%s;%s;Bearer %s;%s", pathname, sha1Hash, requestID, d.AccessToken, timestamp)
	signature := strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(encStr))))

	req.SetHeaders(map[string]string{
		"Accept":        "application/json, text/plain, */*",
		"Authorization": "Bearer " + d.AccessToken,
		"Content-Type":  "application/json",
		"X-User-Agent":  "Web|Chrome 127.0.0.0||OS X|homecloudWebDisk_1.1.1||yunpan 1.1.1|unknown",
		"Timestamp":     timestamp,
		"Signature":     signature,
		"Request-Id":    requestID,
		"userId":        d.UserID,
	})

	var e FrontResp
	req.SetResult(&e)
	res, err := req.Execute(method, url)

	//log.Debugln(res.String())
	if e.Ret != 200 {
		return nil, errors.New(e.Reason)
	}
	if resp != nil {
		err = utils.Json.Unmarshal(res.Body(), resp)
		if err != nil {
			return nil, err
		}
	}
	//fmt.Println(string(res.Body()))
	return res.Body(), nil
}

func (d *HomeCloud) post(pathname string, data interface{}, resp interface{}) ([]byte, error) {
	return d.request(pathname, http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, resp)
}

func (d *HomeCloud) getFiles(catalogID string) ([]model.Obj, error) {
	start := 0
	limit := 100
	files := make([]model.Obj, 0)
	for {
		data := base.Json{
			"catalogID":       catalogID,
			"sortDirection":   1,
			"startNumber":     start + 1,
			"endNumber":       start + limit,
			"filterType":      0,
			"catalogSortType": 0,
			"contentSortType": 0,
			"commonAccountInfo": base.Json{
				"account":     d.Account,
				"accountType": 1,
			},
		}
		var resp GetDiskResp
		_, err := d.post("/orchestration/personalCloud/catalog/v1.0/getDisk", data, &resp)
		if err != nil {
			return nil, err
		}
		for _, catalog := range resp.Data.GetDiskResult.CatalogList {
			f := model.Object{
				ID:       catalog.CatalogID,
				Name:     catalog.CatalogName,
				Size:     0,
				Modified: getTime(catalog.UpdateTime),
				Ctime:    getTime(catalog.CreateTime),
				IsFolder: true,
			}
			files = append(files, &f)
		}
		for _, content := range resp.Data.GetDiskResult.ContentList {
			f := model.ObjThumb{
				Object: model.Object{
					ID:       content.ContentID,
					Name:     content.ContentName,
					Size:     content.ContentSize,
					Modified: getTime(content.UpdateTime),
					HashInfo: utils.NewHashInfo(utils.MD5, content.Digest),
				},
				Thumbnail: model.Thumbnail{Thumbnail: content.ThumbnailURL},
				//Thumbnail: content.BigthumbnailURL,
			}
			files = append(files, &f)
		}
		if start+limit >= resp.Data.GetDiskResult.NodeCount {
			break
		}
		start += limit
	}
	return files, nil
}

func (d *HomeCloud) newJson(data map[string]interface{}) base.Json {
	common := map[string]interface{}{}
	return utils.MergeMap(data, common)
}

func (d *HomeCloud) familyGetFiles(catalogID string) ([]model.Obj, error) {

	// if strings.Contains(catalogID, "/") {
	// 	catalogID = "0"
	// }

	pageNum := 1
	files := make([]model.Obj, 0)
	for {
		data := base.Json{
			"pageInfo": base.Json{
				"pageNum":  pageNum,
				"pageSize": 100,
			},
			"sortInfo": base.Json{
				"sortField": 1,
				"sortOrder": 2,
			},
			"userId":  d.UserID,
			"groupId": d.GroupID,
			"fileId":  catalogID,
		}

		//https://homecloud.komect.com/front/storage/getFileInfoList/v1
		var resp QueryContentListResp
		_, err := d.post("/storage/getFileInfoList/v1", data, &resp)
		if err != nil {
			return nil, err
		}
		for _, content := range resp.Data.FileInfos {
			filesize, err := strconv.ParseInt(content.Size, 10, 64)

			if err != nil {
				return nil, err
			}

			isfolder := false

			if content.Type == 1 {
				isfolder = true
			}

			ctimestamp, err := strconv.ParseInt(content.CreateTime, 10, 64)
			if err != nil {
				fmt.Println("Error parsing timestamp:", err)
				return nil, err
			}

			mtimestamp, err := strconv.ParseInt(content.UpdateTime, 10, 64)
			if err != nil {
				fmt.Println("Error parsing timestamp:", err)
				return nil, err
			}

			// 转换为秒和纳秒
			cseconds := ctimestamp / 1000
			cnanoseconds := (ctimestamp % 1000) * 1000000

			// 转换为秒和纳秒
			mseconds := mtimestamp / 1000
			mnanoseconds := (mtimestamp % 1000) * 1000000

			// 创建 time.Time 对象
			ct := time.Unix(cseconds, cnanoseconds)
			mt := time.Unix(mseconds, mnanoseconds)

			f := model.ObjThumb{
				Object: model.Object{
					ID:       content.ID,
					Name:     content.Name,
					Size:     filesize,
					IsFolder: isfolder,
					Modified: mt,
					Ctime:    ct,
				},
			}
			files = append(files, &f)
		}

		total_count, err := strconv.Atoi(resp.Data.Total)

		if err != nil {
			return nil, err
		}

		if 100*pageNum > total_count {
			break
		}
		pageNum++
	}
	return files, nil
}

func (d *HomeCloud) getLink(contentId string) (string, error) {
	data := base.Json{
		"userId":  d.UserID,
		"groupId": d.GroupID,
		"fileId":  contentId,
	}

	res, err := d.post("/storage/getFileDownloadUrl/v1",
		data, nil)
	if err != nil {
		return "", err
	}
	download_url := "https://cdn.homecloud.komect.com/gateway" + jsoniter.Get(res, "data", "downloadUrl").ToString()
	return download_url, nil
}

func unicode(str string) string {
	textQuoted := strconv.QuoteToASCII(str)
	textUnquoted := textQuoted[1 : len(textQuoted)-1]
	return textUnquoted
}

func (d *HomeCloud) personalRequest(pathname string, method string, callback base.ReqCallback, resp interface{}) ([]byte, error) {
	url := "https://personal-kd-njs.yun.139.com" + pathname
	req := base.RestyClient.R()
	randStr := random.String(16)
	ts := time.Now().Format("2006-01-02 15:04:05")
	if callback != nil {
		callback(req)
	}
	body, err := utils.Json.Marshal(req.Body)
	if err != nil {
		return nil, err
	}
	sign := calSign(string(body), ts, randStr)
	svcType := "1"
	if d.isFamily() {
		svcType = "2"
	}
	req.SetHeaders(map[string]string{
		"Accept":               "application/json, text/plain, */*",
		"Authorization":        "Basic " + d.AccessToken,
		"Caller":               "web",
		"Cms-Device":           "default",
		"Mcloud-Channel":       "1000101",
		"Mcloud-Client":        "10701",
		"Mcloud-Route":         "001",
		"Mcloud-Sign":          fmt.Sprintf("%s,%s,%s", ts, randStr, sign),
		"Mcloud-Version":       "7.13.0",
		"Origin":               "https://yun.139.com",
		"Referer":              "https://yun.139.com/w/",
		"x-DeviceInfo":         "||9|7.13.0|chrome|120.0.0.0|||windows 10||zh-CN|||",
		"x-huawei-channelSrc":  "10000034",
		"x-inner-ntwk":         "2",
		"x-m4c-caller":         "PC",
		"x-m4c-src":            "10002",
		"x-SvcType":            svcType,
		"X-Yun-Api-Version":    "v1",
		"X-Yun-App-Channel":    "10000034",
		"X-Yun-Channel-Source": "10000034",
		"X-Yun-Client-Info":    "||9|7.13.0|chrome|120.0.0.0|||windows 10||zh-CN|||dW5kZWZpbmVk||",
		"X-Yun-Module-Type":    "100",
		"X-Yun-Svc-Type":       "1",
	})

	var e BaseResp
	req.SetResult(&e)
	res, err := req.Execute(method, url)
	if err != nil {
		return nil, err
	}
	log.Debugln(res.String())
	if !e.Success {
		return nil, errors.New(e.Message)
	}
	if resp != nil {
		err = utils.Json.Unmarshal(res.Body(), resp)
		if err != nil {
			return nil, err
		}
	}
	return res.Body(), nil
}
func (d *HomeCloud) personalPost(pathname string, data interface{}, resp interface{}) ([]byte, error) {
	return d.personalRequest(pathname, http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, resp)
}

func getPersonalTime(t string) time.Time {
	stamp, err := time.ParseInLocation("2006-01-02T15:04:05.999-07:00", t, utils.CNLoc)
	if err != nil {
		panic(err)
	}
	return stamp
}

func (d *HomeCloud) personalGetFiles(fileId string) ([]model.Obj, error) {
	files := make([]model.Obj, 0)
	nextPageCursor := ""
	for {
		data := base.Json{
			"imageThumbnailStyleList": []string{"Small", "Large"},
			"orderBy":                 "updated_at",
			"orderDirection":          "DESC",
			"pageInfo": base.Json{
				"pageCursor": nextPageCursor,
				"pageSize":   100,
			},
			"parentFileId": fileId,
		}
		var resp PersonalListResp
		_, err := d.personalPost("/hcy/file/list", data, &resp)
		if err != nil {
			return nil, err
		}
		nextPageCursor = resp.Data.NextPageCursor
		for _, item := range resp.Data.Items {
			var isFolder = (item.Type == "folder")
			var f model.Obj
			if isFolder {
				f = &model.Object{
					ID:       item.FileId,
					Name:     item.Name,
					Size:     0,
					Modified: getPersonalTime(item.UpdatedAt),
					Ctime:    getPersonalTime(item.CreatedAt),
					IsFolder: isFolder,
				}
			} else {
				var Thumbnails = item.Thumbnails
				var ThumbnailUrl string
				if len(Thumbnails) > 0 {
					ThumbnailUrl = Thumbnails[len(Thumbnails)-1].Url
				}
				f = &model.ObjThumb{
					Object: model.Object{
						ID:       item.FileId,
						Name:     item.Name,
						Size:     item.Size,
						Modified: getPersonalTime(item.UpdatedAt),
						Ctime:    getPersonalTime(item.CreatedAt),
						IsFolder: isFolder,
					},
					Thumbnail: model.Thumbnail{Thumbnail: ThumbnailUrl},
				}
			}
			files = append(files, f)
		}
		if len(nextPageCursor) == 0 {
			break
		}
	}
	return files, nil
}

func (d *HomeCloud) personalGetLink(fileId string) (string, error) {
	data := base.Json{
		"fileId": fileId,
	}
	res, err := d.personalPost("/hcy/file/getDownloadUrl",
		data, nil)
	if err != nil {
		return "", err
	}
	var cdnUrl = jsoniter.Get(res, "data", "cdnUrl").ToString()
	if cdnUrl != "" {
		return cdnUrl, nil
	} else {
		return jsoniter.Get(res, "data", "url").ToString(), nil
	}
}
