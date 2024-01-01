package cmd

import (
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/message"
	"github.com/tencent-connect/botgo/log"
	"strconv"
	"time"
)

func SendReply(ctx *Context, msg *dto.MessageToCreate) {
	if ctx.Data.DirectMessage {
		if _, err := (*ctx.Api).PostDirectMessage(ctx, &dto.DirectMessage{
			GuildID:    ctx.Data.GuildID,
			ChannelID:  ctx.Data.ChannelID,
			CreateTime: strconv.FormatInt(time.Now().Unix(), 10),
		}, msg); err != nil {
			log.Error(err)
		}
	} else {
		msg.Content = message.MentionUser(ctx.Data.Author.ID) + "\n" + msg.Content
		if _, err := (*ctx.Api).PostMessage(ctx, ctx.Data.ChannelID, msg); err != nil {
			log.Error(err)
		}
	}
}

func SendReplyNotAt(ctx *Context, msg *dto.MessageToCreate) {
	if ctx.Data.DirectMessage {
		if _, err := (*ctx.Api).PostDirectMessage(ctx, &dto.DirectMessage{
			GuildID:    ctx.Data.GuildID,
			ChannelID:  ctx.Data.ChannelID,
			CreateTime: strconv.FormatInt(time.Now().Unix(), 10),
		}, msg); err != nil {
			log.Error(err)
		}
	} else {
		if _, err := (*ctx.Api).PostMessage(ctx, ctx.Data.ChannelID, msg); err != nil {
			log.Error(err)
		}
	}
}
