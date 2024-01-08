package dialog

import cmd "github.com/Fndream/go-qq-cmd"

// 确认取消框
const (
	YES = iota
	NO
)

var YesNoOptionView = "【确定】【取消】"

type yesNoDialog struct {
	*cmd.BaseDialog
}

func (d *yesNoDialog) Handle(ctx *cmd.Context) interface{} {
	switch ctx.Msg {
	case "确定", "Yes", "yes":
		return YES
	case "取消", "No", "no":
		return NO
	}
	return -1
}

func WaitYesNoDialog(ctx *cmd.Context, msgView *cmd.MsgView) int {
	var dialog cmd.Dialog = &yesNoDialog{
		BaseDialog: &cmd.BaseDialog{
			MainMsgView: msgView,
			Channel:     make(chan *cmd.Context),
			NoChannel:   ctx.Cmd.NoChannel,
			NoDirect:    ctx.Cmd.NoDirect,
		},
	}
	result := cmd.WaitDialog(dialog, ctx)
	return result.(int)
}
