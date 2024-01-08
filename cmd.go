package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
	"reflect"
	"strings"
)

const spaceCharSet = " \u00A0"

type MsgView struct {
	Msg   string
	NotAt bool
}

type Context struct {
	context.Context
	Api     openapi.OpenAPI // api
	Data    *dto.Message    // 事件数据
	Direct  bool            // 是否是私聊事件
	Msg     string          // 消息内容
	Cmd     *Command        // 指令
	CmdName string          // 指令名
	Args    []string        // 参数
}

type Command struct {
	*Config
	Handles []interface{}
}

type Config struct {
	ID          string   // ID
	Name        string   // 名称
	Alias       []string // 别名
	Usage       string   // 用法
	Emoji       string   // emoji图标
	Description string   // 描述
	NoChannel   bool     // 是否在频道中不可用
	NoDirect    bool     // 是否在私信中不可用
	Private     bool     // 是否内部指令
	Async       bool     // 是否可异步执行
}

var idMap = make(map[string]*Command)
var nameMap = make(map[string]*Command)
var privateMap = make(map[string]*Command)

var api openapi.OpenAPI

func SetApi(i openapi.OpenAPI) {
	api = i
}

func Register(config *Config, handles ...interface{}) {
	cmd := Command{
		Config:  config,
		Handles: handles,
	}

	if config.Private {
		privateMap[config.ID] = &cmd
		return
	}
	if config.ID != "" {
		idMap[config.ID] = &cmd
	}
	if config.Name != "" {
		nameMap[config.Name] = &cmd
	}
	for _, alias := range config.Alias {
		nameMap[alias] = &cmd
	}
}

func Process(data *dto.Message) {
	msg := strings.Trim(data.Content, spaceCharSet)
	if msg == "" {
		return
	}

	msgArgs := parseMessageArgs(msg)
	var cmdName string
	var cmdArgs []string
	if msgArgs[0][0] != '<' {
		if msgArgs[0][0] == '/' {
			cmdName = msgArgs[0][1:]
		} else {
			cmdName = msgArgs[0]
		}
		cmdArgs = msgArgs[1:]
	} else {
		if msgArgs[1][0] == '/' {
			cmdName = msgArgs[1][1:]
		} else {
			cmdName = msgArgs[1]
		}
		cmdArgs = msgArgs[2:]
	}

	ctx := &Context{
		Context: context.Background(),
		Api:     api,
		Data:    data,
		Direct:  data.DirectMessage,
		Msg:     msg,
		CmdName: cmdName,
		Args:    cmdArgs,
	}

	cmd, cmdOk := nameMap[cmdName]
	ds, dlOk := userDialogs.Load(ctx.Data.Author.ID)

	if dlOk {
		dl := ds.(*DialogStack).Last()
		// 如果对话框环境和当前消息环境一致
		if (!ctx.Direct && !dl.IsNoChannel()) || (ctx.Direct && !dl.IsNoDirect()) {
			if cmdOk {
				dl.SendMainMsgView(ctx)
				return
			}
			ctx.Cmd = &Command{
				Config: &Config{
					NoChannel: dl.IsNoChannel(),
					NoDirect:  dl.IsNoDirect(),
				},
				Handles: nil,
			}
			dl.GetChannel() <- ctx
			return
		} else {
			SendReply(ctx, &MsgView{
				Msg: "💬 还有未处理完成的对话",
			})
			return
		}
	}

	if !cmdOk {
		return
	}

	ctx.Cmd = cmd

	if (!ctx.Direct && cmd.NoChannel) || (ctx.Direct && cmd.NoDirect) {
		return
	}

	Run(ctx)
	return
}

func GetPrivateCommand(id string) (*Command, bool) {
	cmd, ok := privateMap[id]
	return cmd, ok
}

func Run(ctx *Context) {
	defer func() {
		if er := recover(); er != nil {
			if s, ok := er.(string); ok {
				errorHandle(ctx, errors.New(s))
			} else if e, ok := er.(error); ok {
				errorHandle(ctx, e)
			}
		}
	}()

	handle, params, err := findHandle(ctx)
	if err != nil {
		errorHandle(ctx, err)
		return
	}

	SendRunning(&RunningCommand{
		Ctx:    ctx,
		Handle: handle,
		Params: params,
	})
	return
}

func findHandle(ctx *Context) (handle interface{}, params []reflect.Value, err error) {
	var handles []interface{}
	if ctx.Cmd.Private {
		cmd, ok := privateMap[ctx.Cmd.ID]
		if !ok {
			err = errors.New(fmt.Sprintf("Cannot find %v command GetChannel", ctx.Cmd.ID))
			return
		}
		handles = cmd.Handles
	} else {
		cmd, ok := idMap[ctx.Cmd.ID]
		if !ok {
			err = errors.New(fmt.Sprintf("Cannot find %v command GetChannel", ctx.Cmd.ID))
			return
		}
		handles = cmd.Handles
	}

handle:
	for _, handle := range handles {
		handleType := reflect.TypeOf(handle)

		invokeParams := make([]reflect.Value, 0, handleType.NumIn())
		invokeParams = append(invokeParams, reflect.ValueOf(ctx))

		for j := 1; j < handleType.NumIn(); j++ {
			paramType := handleType.In(j)
			if j-1 >= len(ctx.Args) || ctx.Args[j-1] == "_" {
				if paramType.Kind() != reflect.Pointer {
					continue handle
				}
				invokeParams = append(invokeParams, reflect.New(paramType).Elem())
			} else {
				var val interface{}
				if paramType.Kind() == reflect.Pointer {
					v, er := convArg(ctx.Args[j-1], paramType.Elem())
					if er != nil {
						continue handle
					}
					val = v
				} else {
					v, er := convArg(ctx.Args[j-1], paramType)
					if er != nil {
						continue handle
					}
					val = v
				}

				if paramType.Kind() == reflect.Pointer {
					v := reflect.New(paramType.Elem())
					v.Elem().Set(reflect.ValueOf(val))
					invokeParams = append(invokeParams, v)
				} else {
					invokeParams = append(invokeParams, reflect.ValueOf(val))
				}
			}
		}
		return handle, invokeParams, nil
	}
	msg := "⚠ 参数格式错误"
	if ctx.Cmd.Config.Usage != "" {
		msg += "\n❓ 用法：" + ctx.Cmd.Usage
	}
	err = errors.New(msg)
	return
}
