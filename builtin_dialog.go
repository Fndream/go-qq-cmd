package cmd

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
