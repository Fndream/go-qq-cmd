package cmd

import (
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/message"
	"github.com/tencent-connect/botgo/log"
	"strconv"
	"time"
)

func SendReply(ctx *Context, msg *MsgView) {
	if ctx.Data.DirectMessage {
		if _, err := ctx.Api.PostDirectMessage(ctx, &dto.DirectMessage{
			GuildID:    ctx.Data.GuildID,
			ChannelID:  ctx.Data.ChannelID,
			CreateTime: strconv.FormatInt(time.Now().Unix(), 10),
		}, &dto.MessageToCreate{
			Content: msg.Msg,
			MsgID:   ctx.Data.ID,
		}); err != nil {
			log.Error(err)
		}
	} else {
		if !msg.NotAt {
			msg.Msg = message.MentionUser(ctx.Data.Author.ID) + "\n" + msg.Msg
		}
		if _, err := ctx.Api.PostMessage(ctx, ctx.Data.ChannelID, &dto.MessageToCreate{
			Content: msg.Msg,
			MsgID:   ctx.Data.ID,
		}); err != nil {
			log.Error(err)
		}
	}
}
