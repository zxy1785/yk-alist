package bootstrap

import (
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/fs"
	"github.com/alist-org/alist/v3/internal/offline_download/tool"
	"github.com/alist-org/alist/v3/pkg/tache"
)

func InitTaskManager() {
	fs.UploadTaskManager = tache.NewManager[*fs.UploadTask](tache.WithWorks(conf.Conf.Tasks.Upload.Workers), tache.WithPersistPath(conf.Conf.Tasks.Upload.PersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Upload.MaxRetry))
	fs.CopyTaskManager = tache.NewManager[*fs.CopyTask](tache.WithWorks(conf.Conf.Tasks.Copy.Workers), tache.WithPersistPath(conf.Conf.Tasks.Copy.PersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Copy.MaxRetry))
	tool.DownloadTaskManager = tache.NewManager[*tool.DownloadTask](tache.WithWorks(conf.Conf.Tasks.Download.Workers), tache.WithPersistPath(conf.Conf.Tasks.Download.PersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Download.MaxRetry))
	tool.TransferTaskManager = tache.NewManager[*tool.TransferTask](tache.WithWorks(conf.Conf.Tasks.Transfer.Workers), tache.WithPersistPath(conf.Conf.Tasks.Transfer.PersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Transfer.MaxRetry))
}
