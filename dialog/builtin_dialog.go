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

func WaitYesNoDialog(ctx *cmd.Context, msg *cmd.MsgView) int {
	var dialog cmd.Dialog = &yesNoDialog{
		BaseDialog: &cmd.BaseDialog{
			Ctx:         ctx,
			MainMsgView: msg,
			Channel:     make(chan *cmd.Context),
		},
	}
	result := cmd.WaitDialog(dialog)
	return result.(int)
}

func WaitYesNoDialogS(ctx *cmd.Context, msg string) int {
	return WaitYesNoDialog(ctx, &cmd.MsgView{Msg: msg})
}
