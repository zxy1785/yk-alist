package tool

import (
	"fmt"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/tache"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type DownloadTask struct {
	tache.Base
	Name         string       `json:"name"`
	Url          string       `json:"url"`
	DstDirPath   string       `json:"dst_dir_path"`
	TempDir      string       `json:"temp_dir"`
	DeletePolicy DeletePolicy `json:"delete_policy"`

	Status            string   `json:"status"`
	Signal            chan int `json:"-"`
	GID               string   `json:"-"`
	tool              Tool
	callStatusRetried int
}

func (t *DownloadTask) OnFailed() {
	result := fmt.Sprintf("%s下载失败:%s", t.Url, t.GetErr())
	log.Debug(result)
	if setting.GetBool(conf.NotifyEnabled) && setting.GetBool(conf.NotifyOnDownloadFailed) {
		go op.Notify("文件下载结果", result)
	}

}

func (t *DownloadTask) OnSucceeded() {
	result := fmt.Sprintf("%s下载成功", t.Url)
	log.Debug(result)
	if setting.GetBool(conf.NotifyEnabled) && setting.GetBool(conf.NotifyOnDownloadSucceeded) {
		go op.Notify("文件下载结果", result)
	}
}

func (t *DownloadTask) Run() error {
	t.Name = fmt.Sprintf("download %s to (%s)", t.Url, t.DstDirPath)
	if err := t.tool.Run(t); !errs.IsNotSupportError(err) {
		if err == nil {
			return t.Complete()
		}
		return err
	}
	t.Signal = make(chan int)
	defer func() {
		t.Signal = nil
	}()
	gid, err := t.tool.AddURL(&AddUrlArgs{
		Url:     t.Url,
		UID:     t.ID,
		TempDir: t.TempDir,
		Signal:  t.Signal,
	})
	if err != nil {
		return err
	}
	t.GID = gid
	var (
		ok bool
	)
outer:
	for {
		select {
		case <-t.CtxDone():
			err := t.tool.Remove(t)
			return err
		case <-t.Signal:
			ok, err = t.Update()
			if ok {
				break outer
			}
		case <-time.After(time.Second * 3):
			ok, err = t.Update()
			if ok {
				break outer
			}
		}
	}
	if err != nil {
		return err
	}
	t.Status = "offline download completed, maybe transferring"
	// hack for qBittorrent
	if t.tool.Name() == "qBittorrent" {
		seedTime := setting.GetInt(conf.QbittorrentSeedtime, 0)
		if seedTime >= 0 {
			t.Status = "offline download completed, waiting for seeding"
			<-time.After(time.Minute * time.Duration(seedTime))
			err := t.tool.Remove(t)
			if err != nil {
				log.Errorln(err.Error())
			}
		}
	}
	return nil
}

// Update download status, return true if download completed
func (t *DownloadTask) Update() (bool, error) {
	info, err := t.tool.Status(t)
	if err != nil {
		t.callStatusRetried++
		log.Errorf("failed to get status of %s, retried %d times", t.ID, t.callStatusRetried)
		return false, nil
	}
	if t.callStatusRetried > 5 {
		return true, errors.Errorf("failed to get status of %s, retried %d times", t.ID, t.callStatusRetried)
	}
	t.callStatusRetried = 0
	t.SetProgress(info.Progress)
	t.Status = fmt.Sprintf("[%s]: %s", t.tool.Name(), info.Status)
	if info.NewGID != "" {
		log.Debugf("followen by: %+v", info.NewGID)
		t.GID = info.NewGID
		return false, nil
	}
	// if download completed
	if info.Completed {
		err := t.Complete()
		return true, errors.WithMessage(err, "failed to transfer file")
	}
	// if download failed
	if info.Err != nil {
		return true, errors.Errorf("failed to download %s, error: %s", t.ID, info.Err.Error())
	}
	return false, nil
}

func (t *DownloadTask) Complete() error {
	var (
		files []File
		err   error
	)
	if getFileser, ok := t.tool.(GetFileser); ok {
		files = getFileser.GetFiles(t)
	} else {
		files, err = GetFiles(t.TempDir)
		if err != nil {
			return errors.Wrapf(err, "failed to get files")
		}
	}
	// upload files
	for i, _ := range files {
		file := files[i]
		TransferTaskManager.Add(&TransferTask{
			file:         file,
			DstDirPath:   t.DstDirPath,
			TempDir:      t.TempDir,
			DeletePolicy: t.DeletePolicy,
			FileDir:      file.Path,
		})
	}
	return nil
}

func (t *DownloadTask) GetName() string {
	return t.Name
	//return fmt.Sprintf("download %s to (%s)", t.Url, t.DstDirPath)
}

func (t *DownloadTask) GetStatus() string {
	return t.Status
}

var (
	DownloadTaskManager *tache.Manager[*DownloadTask]
)
