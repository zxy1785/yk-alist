package bootstrap

import (
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/fs"
	"github.com/alist-org/alist/v3/internal/offline_download/tool"
	"github.com/alist-org/alist/v3/pkg/tache"
)

func InitTaskManager() {
	fs.UploadTaskManager = tache.NewManager[*fs.UploadTask](tache.WithWorks(conf.Conf.Tasks.Upload.Workers), tache.WithMaxRetry(conf.Conf.Tasks.Upload.MaxRetry)) //upload will not support persist
	fs.CopyTaskManager = tache.NewManager[*fs.CopyTask](tache.WithWorks(conf.Conf.Tasks.Copy.Workers), tache.WithPersistFunction(db.GetTaskDataFunc("copy", conf.Conf.Tasks.Copy.TaskPersistant), db.UpdateTaskDataFunc("copy", conf.Conf.Tasks.Copy.TaskPersistant)), tache.WithMaxRetry(conf.Conf.Tasks.Copy.MaxRetry))
	tool.DownloadTaskManager = tache.NewManager[*tool.DownloadTask](tache.WithWorks(conf.Conf.Tasks.Download.Workers), tache.WithPersistFunction(db.GetTaskDataFunc("download", conf.Conf.Tasks.Download.TaskPersistant), db.UpdateTaskDataFunc("download", conf.Conf.Tasks.Download.TaskPersistant)), tache.WithMaxRetry(conf.Conf.Tasks.Download.MaxRetry))
	tool.TransferTaskManager = tache.NewManager[*tool.TransferTask](tache.WithWorks(conf.Conf.Tasks.Transfer.Workers), tache.WithPersistFunction(db.GetTaskDataFunc("transfer", conf.Conf.Tasks.Transfer.TaskPersistant), db.UpdateTaskDataFunc("transfer", conf.Conf.Tasks.Transfer.TaskPersistant)), tache.WithMaxRetry(conf.Conf.Tasks.Transfer.MaxRetry))
	if len(tool.TransferTaskManager.GetAll()) == 0 { //prevent offline downloaded files from being deleted
		CleanTempDir()
	}
}

// func InitTaskManager() {

// 	uploadTaskPersistPath := conf.Conf.Tasks.Upload.PersistPath
// 	copyTaskPersistPath := conf.Conf.Tasks.Copy.PersistPath
// 	downloadTaskPersistPath := conf.Conf.Tasks.Download.PersistPath
// 	transferTaskPersistPath := conf.Conf.Tasks.Transfer.PersistPath
// 	if !utils.Exists(uploadTaskPersistPath) {
// 		log.Infof("传输任务持久化文件")
// 		_, err := utils.CreateNestedFile(uploadTaskPersistPath)
// 		if err != nil {
// 			log.Fatalf("创建上传任务文件失败: %+v", err)
// 		}
// 	}

// 	if !utils.Exists(copyTaskPersistPath) {
// 		log.Infof("复制任务持久化文件")
// 		_, err := utils.CreateNestedFile(copyTaskPersistPath)
// 		if err != nil {
// 			log.Fatalf("创建复制任务文件失败: %+v", err)
// 		}

// 	}

// 	if !utils.Exists(downloadTaskPersistPath) {
// 		log.Infof("下载任务持久化文件")
// 		_, err := utils.CreateNestedFile(downloadTaskPersistPath)
// 		if err != nil {
// 			log.Fatalf("创建下载任务文件失败: %+v", err)
// 		}
// 	}

// 	if !utils.Exists(transferTaskPersistPath) {
// 		log.Infof("传输任务持久化文件")
// 		_, err := utils.CreateNestedFile(transferTaskPersistPath)
// 		if err != nil {
// 			log.Fatalf("创建传输任务文件失败: %+v", err)
// 		}
// 	}

// 	fs.UploadTaskManager = tache.NewManager[*fs.UploadTask](tache.WithWorks(conf.Conf.Tasks.Upload.Workers), tache.WithPersistPath(uploadTaskPersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Upload.MaxRetry))
// 	fs.CopyTaskManager = tache.NewManager[*fs.CopyTask](tache.WithWorks(conf.Conf.Tasks.Copy.Workers), tache.WithPersistPath(copyTaskPersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Copy.MaxRetry))
// 	tool.DownloadTaskManager = tache.NewManager[*tool.DownloadTask](tache.WithWorks(conf.Conf.Tasks.Download.Workers), tache.WithPersistPath(downloadTaskPersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Download.MaxRetry))
// 	tool.TransferTaskManager = tache.NewManager[*tool.TransferTask](tache.WithWorks(conf.Conf.Tasks.Transfer.Workers), tache.WithPersistPath(transferTaskPersistPath), tache.WithMaxRetry(conf.Conf.Tasks.Transfer.MaxRetry))
// }
