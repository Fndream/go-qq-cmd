package cmd

import (
	"errors"
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

type userChannel struct {
	channel chan *RunningCommand
	direct  chan *RunningCommand
}

func SendRunning(running *RunningCommand) {
	uid := running.Ctx.Data.Author.ID
	ch, loaded := userChannels.LoadOrStore(uid, &userChannel{
		channel: make(chan *RunningCommand, 16),
		direct:  make(chan *RunningCommand, 16),
	})
	if !loaded {
		go func() {
			for {
				select {
				case rc := <-ch.(*userChannel).channel:
					callHandle(rc)
				case rc := <-ch.(*userChannel).direct:
					callHandle(rc)
				case <-time.After(5 * time.Minute):
					userChannels.Delete(uid)
					close(ch.(chan *RunningCommand))
					return
				}
			}
		}()
	}
	uc := ch.(*userChannel)
	if running.Ctx.Direct {
		uc.direct <- running
	} else {
		uc.channel <- running
	}
}

func callHandle(rc *RunningCommand) {
	defer func() {
		if er := recover(); er != nil {
			if s, ok := er.(string); ok {
				errorHandle(rc.Ctx, errors.New(s))
			} else if e, ok := er.(error); ok {
				errorHandle(rc.Ctx, e)
			}
		}
	}()
	var err interface{}
	retryCount := 0
	for retryCount <= 3 {
		r := reflect.ValueOf(rc.Handle).Call(rc.Params)
		resultHandle(rc.Ctx, r[0].Interface().(*MsgView))
		err = r[1].Interface()
		if err == nil {
			break
		}
		if errors.Is(err.(error), RetryError) {
			retryCount++
			continue
		}
		errorHandle(rc.Ctx, err.(error))
		break
	}
}

func resultHandle(ctx *Context, msgView *MsgView) {
	if msgView != nil {
		SendReply(ctx, msgView)
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
		SendReply(ctx, &MsgView{
			Msg: msg,
		})
	}
}
