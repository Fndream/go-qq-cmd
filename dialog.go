package cmd

import (
	"sync"
)

var dialogs = sync.Map{}

type Dialog interface {
	GetMainMsgView() *MsgView
	GetChannel() chan *Context
	SendMainMsgView(ctx *Context)
	Handle(ctx *Context) interface{}
}

type BaseDialog struct {
	MainMsgView *MsgView      // 主消息视图
	Channel     chan *Context // 通道，用于回复Dialog
}

func (b *BaseDialog) GetMainMsgView() *MsgView {
	return b.MainMsgView
}

func (b *BaseDialog) GetChannel() chan *Context {
	return b.Channel
}

func (b *BaseDialog) SendMainMsgView(ctx *Context) {
	SendReply(ctx, b.MainMsgView)
}

func (b *BaseDialog) Handle(ctx *Context) interface{} {
	return false
}

func WaitDialog(dialog *Dialog, ctx *Context) interface{} {
	dialogs.Store(ctx.Data.Author.ID, dialog)
	(*dialog).SendMainMsgView(ctx)
	for {
		c := (*dialog).GetChannel()
		x := <-c
		r := (*dialog).Handle(x)
		if r, ok := r.(int); !ok || r != -1 {
			close(c)
			dialogs.Delete(ctx.Data.Author.ID)
			return r
		}
	}
}

// 确认取消框

const (
	YES = iota
	NO
)

type yesNoDialog struct {
	BaseDialog
}

func (d *yesNoDialog) Handle(ctx *Context) interface{} {
	switch ctx.Msg {
	case "确定", "Yes", "yes":
		return YES
	case "取消", "No", "no":
		return NO
	}
	return -1
}

func WaitYesNoDialog(ctx *Context, msgView *MsgView) int {
	var dialog Dialog = &yesNoDialog{
		BaseDialog: BaseDialog{
			MainMsgView: msgView,
			Channel:     make(chan *Context),
		},
	}
	result := WaitDialog(&dialog, ctx)
	return result.(int)
}
