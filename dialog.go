package cmd

import (
	"sync"
)

var dialogs = sync.Map{}

type Dialog interface {
	SendMsgView(ctx *Context)
	Channel() chan *Context
}

type BaseDialog struct {
	MsgView *MsgView
	channel chan *Context
}

func (b *BaseDialog) SendMsgView(ctx *Context) {
	SendReply(ctx, b.MsgView)
}

func (b *BaseDialog) Channel() chan *Context {
	return b.channel
}

const (
	YES = iota
	NO
)

type yesNoDialog struct {
	BaseDialog
}

func (d yesNoDialog) handleMsg(ctx *Context) int {
	switch ctx.CmdName {
	case "确定", "Yes", "yes":
		v, loaded := dialogs.LoadAndDelete(ctx.Data.Author.ID)
		if loaded {
			close(v.(*yesNoDialog).channel)
		}
		return YES
	case "取消", "No", "no":
		v, loaded := dialogs.LoadAndDelete(ctx.Data.Author.ID)
		if loaded {
			close(v.(*yesNoDialog).channel)
		}
		return NO
	}
	return -1
}

func WaitYesNoDialog(ctx *Context, msg *MsgView) int {
	dl := yesNoDialog{BaseDialog{msg, make(chan *Context)}}
	dialogs.Store(ctx.Data.Author.ID, &dl)
	dl.SendMsgView(ctx)
	i := -1
	for i == -1 {
		c := <-dl.channel
		i = dl.handleMsg(c)
	}
	return i
}
