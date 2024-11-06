package fs

import (
	"context"
	"fmt"
	"net/http"
	stdpath "path"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/alist-org/alist/v3/pkg/tache"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type CopyTask struct {
	tache.Base
	Status       string        `json:"-"` //don't save status to save space
	SrcObjPath   string        `json:"src_path"`
	DstDirPath   string        `json:"dst_path"`
	Override     bool          `json:"override"`
	srcStorage   driver.Driver `json:"-"`
	dstStorage   driver.Driver `json:"-"`
	SrcStorageMp string        `json:"src_storage_mp"`
	DstStorageMp string        `json:"dst_storage_mp"`
}

func (t *CopyTask) GetName() string {
	return fmt.Sprintf("copy [%s](%s) to [%s](%s)", t.SrcStorageMp, t.SrcObjPath, t.DstStorageMp, t.DstDirPath)
}

func (t *CopyTask) GetStatus() string {
	return t.Status
}

func (t *CopyTask) OnFailed() {
	result := fmt.Sprintf("%s:%s", t.GetName(), t.GetErr())
	log.Debug(result)
	go op.Notify("文件复制结果", result)
}

func (t *CopyTask) OnSucceeded() {
	result := fmt.Sprintf("复制%s到%s成功", t.SrcObjPath, t.DstDirPath)
	log.Debug(result)
	go op.Notify("文件复制结果", result)
}

func (t *CopyTask) Run() error {

	var err error
	if t.srcStorage == nil {
		t.srcStorage, err = op.GetStorageByMountPath(t.SrcStorageMp)
	}
	if t.dstStorage == nil {
		t.dstStorage, err = op.GetStorageByMountPath(t.DstStorageMp)
	}
	if err != nil {
		return errors.WithMessage(err, "failed get storage")
	}
	
	if !t.Override {
		srcObj, err := get(context.Background(), t.SrcStorageMp+t.SrcObjPath)
                if err != nil {
		        return errors.WithMessagef(err, "failed get src [%s] file", SrcObjPath)
	        }
	        if srcObj.IsDir() {
		        return copyBetween2Storages(t, t.srcStorage, t.dstStorage, t.SrcObjPath, t.DstDirPath)
	        }
		dst_path := stdpath.Join(t.DstStorageMp+t.DstDirPath, srcObj.GetName())
		obj, err := get(context.Background(), dst_path)
		if err != nil {
			//文件不存在
			return copyBetween2Storages(t, t.srcStorage, t.dstStorage, t.SrcObjPath, t.DstDirPath)
		} else {
			//文件已经存在，直接返回完成
			return errors.WithMessage(err, obj.GetName()+"文件已经存在")
		}
	} else {
		return copyBetween2Storages(t, t.srcStorage, t.dstStorage, t.SrcObjPath, t.DstDirPath)
	}

}

var CopyTaskManager *tache.Manager[*CopyTask]

// Copy if in the same storage, call move method
// if not, add copy task
func _copy(ctx context.Context, SrcObjPath, DstDirPath string, overwrite bool, lazyCache ...bool) (tache.TaskWithInfo, error) {
	srcStorage, srcObjActualPath, err := op.GetStorageAndActualPath(SrcObjPath)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get src storage")
	}
	dstStorage, dstDirActualPath, err := op.GetStorageAndActualPath(DstDirPath)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get dst storage")
	}

	// copy if in the same storage, just call driver.Copy
	if srcStorage.GetStorage() == dstStorage.GetStorage() {
		return nil, op.Copy(ctx, srcStorage, srcObjActualPath, dstDirActualPath, lazyCache...)
	}
	if ctx.Value(conf.NoTaskKey) != nil {
		srcObj, err := op.Get(ctx, srcStorage, srcObjActualPath)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed get src [%s] file", SrcObjPath)
		}
		if !srcObj.IsDir() {
			// copy file directly
			link, _, err := op.Link(ctx, srcStorage, srcObjActualPath, model.LinkArgs{
				Header: http.Header{},
			})
			if err != nil {
				return nil, errors.WithMessagef(err, "failed get [%s] link", SrcObjPath)
			}
			fs := stream.FileStream{
				Obj: srcObj,
				Ctx: ctx,
			}
			// any link provided is seekable
			ss, err := stream.NewSeekableStream(fs, link)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed get [%s] stream", SrcObjPath)
			}
			return nil, op.Put(ctx, dstStorage, dstDirActualPath, ss, nil, false)
		}
	}
	// not in the same storage

	t := &CopyTask{
		srcStorage:   srcStorage,
		dstStorage:   dstStorage,
		SrcObjPath:   srcObjActualPath,
		DstDirPath:   dstDirActualPath,
		Override:     overwrite,
		SrcStorageMp: srcStorage.GetStorage().MountPath,
		DstStorageMp: dstStorage.GetStorage().MountPath,
	}
	CopyTaskManager.Add(t)
	return t, nil
}

func copyBetween2Storages(t *CopyTask, srcStorage, dstStorage driver.Driver, SrcObjPath, DstDirPath string) error {
	t.Status = "getting src object"
	srcObj, err := op.Get(t.Ctx(), srcStorage, SrcObjPath)
	if err != nil {
		return errors.WithMessagef(err, "failed get src [%s] file", SrcObjPath)
	}
	if srcObj.IsDir() {
		t.Status = "src object is dir, listing objs"
		objs, err := op.List(t.Ctx(), srcStorage, SrcObjPath, model.ListArgs{})
		if err != nil {
			return errors.WithMessagef(err, "failed list src [%s] objs", SrcObjPath)
		}
		for _, obj := range objs {
			if utils.IsCanceled(t.Ctx()) {
				return nil
			}
			SrcObjPath := stdpath.Join(SrcObjPath, obj.GetName())
			dstObjPath := stdpath.Join(DstDirPath, srcObj.GetName())
			CopyTaskManager.Add(&CopyTask{
				srcStorage:   srcStorage,
				dstStorage:   dstStorage,
				SrcObjPath:   SrcObjPath,
				DstDirPath:   dstObjPath,
				Override:     t.Override,
				SrcStorageMp: srcStorage.GetStorage().MountPath,
				DstStorageMp: dstStorage.GetStorage().MountPath,
			})
		}
		t.Status = "src object is dir, added all copy tasks of objs"
		return nil
	}
	return copyFileBetween2Storages(t, srcStorage, dstStorage, SrcObjPath, DstDirPath)
}

func copyFileBetween2Storages(tsk *CopyTask, srcStorage, dstStorage driver.Driver, srcFilePath, DstDirPath string) error {
	srcFile, err := op.Get(tsk.Ctx(), srcStorage, srcFilePath)
	if err != nil {
		return errors.WithMessagef(err, "failed get src [%s] file", srcFilePath)
	}
	link, _, err := op.Link(tsk.Ctx(), srcStorage, srcFilePath, model.LinkArgs{
		Header: http.Header{},
	})
	if err != nil {
		return errors.WithMessagef(err, "failed get [%s] link", srcFilePath)
	}
	fs := stream.FileStream{
		Obj: srcFile,
		Ctx: tsk.Ctx(),
	}
	// any link provided is seekable
	ss, err := stream.NewSeekableStream(fs, link)
	if err != nil {
		return errors.WithMessagef(err, "failed get [%s] stream", srcFilePath)
	}
	return op.Put(tsk.Ctx(), dstStorage, DstDirPath, ss, tsk.SetProgress, true)
}
