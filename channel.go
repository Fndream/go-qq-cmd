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

var usersSyncChannel = sync.Map{}

func SendRunning(running *RunningCommand) {
	if running.Ctx.Cmd.Async {
		callHandle(running)
		return
	}
	uid := running.Ctx.Data.Author.ID
	ch, loaded := usersSyncChannel.LoadOrStore(uid, make(chan *RunningCommand, 8))
	if !loaded {
		go func() {
			for {
				select {
				case rc := <-ch.(chan *RunningCommand):
					callHandle(rc)
				case <-time.After(5 * time.Minute):
					close(ch.(chan *RunningCommand))
					usersSyncChannel.Delete(uid)
					return
				}
			}
		}()
	}
	ch.(chan *RunningCommand) <- running
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
