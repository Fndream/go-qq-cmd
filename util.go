package cmd

import (
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/message"
	"github.com/tencent-connect/botgo/log"
)

func SendReply(ctx *Context, msg *dto.MessageToCreate) {
	msg.Content = message.MentionUser(ctx.Data.Author.ID) + "\n" + msg.Content
	if _, err := (*ctx.Api).PostMessage(ctx, ctx.Data.ChannelID, msg); err != nil {
		log.Error(err)
	}
}

func SendReplyNotAt(ctx *Context, msg *dto.MessageToCreate) {
	if _, err := (*ctx.Api).PostMessage(ctx, ctx.Data.ChannelID, msg); err != nil {
		log.Error(err)
	}
}
