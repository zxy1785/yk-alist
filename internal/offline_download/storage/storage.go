package storage

import (
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/offline_download/tool"
	"github.com/pkg/errors"
)

type Storage struct {
}

func (a *Storage) Run(task *tool.DownloadTask) error {
	return errs.NotSupport
}

func (a *Storage) Name() string {
	return "storage"
}

func (a *Storage) Items() []model.SettingItem {
	// qBittorrent settings
	return []model.SettingItem{}
}

func (a *Storage) Init() (string, error) {
	return "ok", nil
}

func (a *Storage) IsReady() bool {
	return true
}

func (a *Storage) AddURL(args *tool.AddUrlArgs) (string, error) {
	return "ok", nil
}

func (a *Storage) Remove(task *tool.DownloadTask) error {
	return errors.Errorf("Failed to Remove")
}

func (a *Storage) Status(task *tool.DownloadTask) (*tool.Status, error) {
	s := &tool.Status{}
	return s, nil
}

var _ tool.Tool = (*Storage)(nil)

func init() {
	tool.Tools.Add(&Storage{})
}
