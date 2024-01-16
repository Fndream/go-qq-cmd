package cmd

import (
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/message"
	"github.com/tencent-connect/botgo/log"
	"qqbot/cmd/cache"
	"strconv"
	"time"
)

type MsgView struct {
	Msg   string
	Image string
	NotAt bool
}

var imageCache = cache.New(10*time.Minute, 5*time.Minute)

func SendReply(ctx *Context, msg *MsgView) {
	imgUrl := ""
	haveCache := false
	if msg.Image != "" {
		cv, ok := imageCache.Get(msg.Image)
		haveCache = ok
		if ok {
			imgUrl = cv.(string)
		} else {
			imgUrl = msg.Image
		}
	}

	msgCreate := &dto.MessageToCreate{
		Image:   imgUrl,
		Content: msg.Msg,
		MsgID:   ctx.Data.ID,
	}

	if ctx.Data.DirectMessage {
		directMessage := dto.DirectMessage{
			GuildID:    ctx.Data.GuildID,
			ChannelID:  ctx.Data.ChannelID,
			CreateTime: strconv.FormatInt(time.Now().Unix(), 10),
		}
		_, err := ctx.Api.PostDirectMessage(ctx, &directMessage, msgCreate)
		if err != nil {
			log.Error(err)
		}
	} else {
		if !msg.NotAt {
			msgCreate.Content = message.MentionUser(ctx.Data.Author.ID) + "\n" + msgCreate.Content
		}
		rep, err := ctx.Api.PostMessage(ctx, ctx.Data.ChannelID, msgCreate)
		if err != nil {
			log.Error(err)
			return
		}
		if imgUrl != "" && !haveCache {
			m, err := ctx.Api.Message(ctx, rep.ChannelID, rep.ID)
			if err != nil {
				log.Error(err)
				return
			}
			imageCache.Set(msg.Image, "https://"+m.Attachments[0].URL)
		}
	}
}

func SendReplyS(ctx *Context, msg string) {
	SendReply(ctx, &MsgView{Msg: msg})
}

func SendReplyJS(ctx *Context, image string, msg string) {
	SendReply(ctx, &MsgView{Msg: msg, Image: image})
}
