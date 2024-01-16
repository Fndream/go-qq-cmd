package cmd

import (
	"sync"
)

var userDialogs = sync.Map{}

type Dialog interface {
	SendMainMsgView(ctx *Context)
	Handle(ctx *Context) interface{}

	GetCtx() *Context
	GetMainMsgView() *MsgView
	GetChannel() chan *Context
}

type BaseDialog struct {
	Ctx         *Context      // 启动此对话框的上下文
	MainMsgView *MsgView      // 主消息视图
	Channel     chan *Context // 通道，用于回复Dialog
}

func (b *BaseDialog) SendMainMsgView(ctx *Context) {
	SendReply(ctx, b.MainMsgView)
}

func (b *BaseDialog) Handle(ctx *Context) interface{} {
	return false
}

func (b *BaseDialog) GetCtx() *Context {
	return b.Ctx
}

func (b *BaseDialog) GetMainMsgView() *MsgView {
	return b.MainMsgView
}

func (b *BaseDialog) GetChannel() chan *Context {
	return b.Channel
}

func WaitDialog(dialog Dialog) interface{} {
	ctx := dialog.GetCtx()
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
