package cmd

import (
	"sync"
)

var userDialogs = sync.Map{}

type Dialog interface {
	SendMainMsgView(ctx *Context)
	Handle(ctx *Context) interface{}

	GetMainMsgView() *MsgView
	GetChannel() chan *Context
	IsNoChannel() bool
	IsNoDirect() bool
}

type BaseDialog struct {
	MainMsgView *MsgView      // 主消息视图
	Channel     chan *Context // 通道，用于回复Dialog
	NoChannel   bool
	NoDirect    bool
}

func (b *BaseDialog) SendMainMsgView(ctx *Context) {
	SendReply(ctx, b.MainMsgView)
}

func (b *BaseDialog) Handle(ctx *Context) interface{} {
	return false
}

func (b *BaseDialog) GetMainMsgView() *MsgView {
	return b.MainMsgView
}

func (b *BaseDialog) GetChannel() chan *Context {
	return b.Channel
}

func (b *BaseDialog) IsNoChannel() bool {
	return b.NoChannel
}

func (b *BaseDialog) IsNoDirect() bool {
	return b.NoDirect
}

func WaitDialog(dialog Dialog, ctx *Context) interface{} {
	stack, _ := userDialogs.LoadOrStore(ctx.Data.Author.ID, &DialogStack{element: []Dialog{}})
	stack.(*DialogStack).Push(dialog)
	dialog.SendMainMsgView(ctx)
	for {
		ch := dialog.GetChannel()
		res := dialog.Handle(<-ch)

		// 如果返回值是-1，此次的回复是无效的，继续循环等待用户重新回复本次对话框
		if r, ok := res.(int); ok && r == -1 {
			continue
		}

		// 对话框正确回复处理，返回结果
		close(ch)
		ds, _ := userDialogs.Load(ctx.Data.Author.ID)
		stack := ds.(*DialogStack)
		stack.Pop()
		if len(stack.element) <= 0 {
			userDialogs.Delete(ctx.Data.Author.ID)
		}
		return res
	}
}

type DialogStack struct {
	element []Dialog
}

func (s *DialogStack) Push(d Dialog) {
	s.element = append(s.element, d)
}

func (s *DialogStack) Last() Dialog {
	return s.element[len(s.element)-1]
}

func (s *DialogStack) Pop() Dialog {
	if len(s.element) == 0 {
		return nil
	}
	index := len(s.element) - 1
	item := s.element[index]
	s.element = s.element[:index]
	return item
}
