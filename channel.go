package cmd

import (
	"errors"
	"github.com/tencent-connect/botgo/dto"
	"reflect"
	"sync"
	"time"
	"unicode/utf8"
)

type RunningCommand struct {
	Ctx    *Context
	Handle interface{}
	Params []reflect.Value
}

var userChannels = sync.Map{}

func SendRunning(running *RunningCommand) {
	uid := running.Ctx.Data.Author.ID
	ch, loaded := userChannels.LoadOrStore(uid, make(chan *RunningCommand, 16))
	if !loaded {
		go func() {
			for {
				select {
				case rc := <-ch.(chan *RunningCommand):
					callHandle(rc)
				case <-time.After(5 * time.Minute):
					userChannels.Delete(uid)
				}
			}
		}()
	}
	ch.(chan *RunningCommand) <- running
}

func callHandle(rc *RunningCommand) {
	r := reflect.ValueOf(rc.Handle).Call(rc.Params)
	resultHandle(rc.Ctx, r[0].Interface().(*Result))
	err := r[1].Interface()

	retryCount := 0
	for retryCount < 3 {
		if err == nil {
			break
		}
		if errors.Is(err.(error), RetryError) {
			r := reflect.ValueOf(rc.Handle).Call(rc.Params)
			resultHandle(rc.Ctx, r[0].Interface().(*Result))
			err = r[1].Interface()
			retryCount++
		} else {
			errorHandle(rc.Ctx, err.(error))
			break
		}
	}
}

func resultHandle(ctx *Context, result *Result) {
	if result != nil {
		if result.NotAt {
			SendReplyNotAt(ctx, &dto.MessageToCreate{
				Content: result.Msg,
				MsgID:   ctx.Data.ID,
			})
		} else {
			SendReply(ctx, &dto.MessageToCreate{
				Content: result.Msg,
				MsgID:   ctx.Data.ID,
			})
		}
	}
}

func errorHandle(ctx *Context, err error) {
	if err != nil {
		var msg string
		r, size := utf8.DecodeRuneInString(err.Error())
		if size > 0 && ((r >= 0x1F300 && r <= 0x1F6FF) || (r >= 0x2600 && r <= 0x26FF)) {
			msg = err.Error()
		} else {
			msg = "❌ " + err.Error()
		}
		SendReply(ctx, &dto.MessageToCreate{
			Content: msg,
			MsgID:   ctx.Data.ID,
		})
	}
}
