package tache

import (
	"context"
	"errors"
)

// Base is the base struct for all tasks to implement TaskBase interface
type Base struct {
	ID       string  `json:"id"`
	State    State   `json:"state"`
	Retry    int     `json:"retry"`
	MaxRetry int     `json:"max_retry"`
	Error    string  `json:"error"`
	Progress float64 `json:"progress"`
	err      error
	ctx      context.Context
	cancel   context.CancelFunc
	persist  func()
}

func (b *Base) SetProgress(progress float64) {
	b.Progress = progress
	b.Persist()
}

func (b *Base) GetProgress() float64 {
	return b.Progress
}

func (b *Base) SetState(state State) {
	b.State = state
	b.Persist()
}

func (b *Base) GetState() State {
	return b.State
}

func (b *Base) GetID() string {
	return b.ID
}

func (b *Base) SetID(id string) {
	b.ID = id
	b.Persist()
}

func (b *Base) SetErr(err error) {
	if err == nil {
		return
	}
	b.err = err
	b.Error = err.Error()
	b.Persist()
}

func (b *Base) GetErr() error {
	return errors.New(b.Error)
	//return b.err
}

func (b *Base) CtxDone() <-chan struct{} {
	return b.Ctx().Done()
}

func (b *Base) SetCtx(ctx context.Context) {
	b.ctx = ctx
}

func (b *Base) SetCancelFunc(cancelFunc context.CancelFunc) {
	b.cancel = cancelFunc
}

func (b *Base) GetRetry() (int, int) {
	return b.Retry, b.MaxRetry
}

func (b *Base) SetRetry(retry int, maxRetry int) {
	b.Retry, b.MaxRetry = retry, maxRetry
}

func (b *Base) Cancel() {
	b.SetState(StateCanceling)
	b.cancel()
}

func (b *Base) Ctx() context.Context {
	return b.ctx
}

func (b *Base) Persist() {
	if b.persist != nil {
		b.persist()
	}
}

func (b *Base) SetPersist(persist func()) {
	b.persist = persist
}

// func (b *Base) OnFailed() {

// }

// func (b *Base) OnSucceeded() {

// }

// func (b *Base) OnBeforeRetry() {

// }

var _ TaskBase = (*Base)(nil)
