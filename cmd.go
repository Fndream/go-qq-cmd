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

type Result struct {
	Msg   string
	NotAt bool
}

type Context struct {
	context.Context
	Api     *openapi.OpenAPI // api
	Data    *dto.Message     // 事件数据
	Msg     string           // 消息内容
	Cmd     *Config          // 指令信息
	CmdName string           // 指令名
	Args    []string         // 参数
}

type Config struct {
	Private     bool     // 是否内部指令
	ID          string   // ID
	Name        string   // 名称
	Alias       []string // 别名
	Usage       string   // 用法
	Emoji       string   // emoji图标
	Description string   // 描述
}

var idConfig = make(map[string]*Config)
var nameConfig = make(map[string]*Config)
var privateConfig = make(map[string]*Config)

var idHandles = make(map[string][]interface{})
var nameHandles = make(map[string][]interface{})
var privateHandles = make(map[string][]interface{})

var api *openapi.OpenAPI

func SetApi(i *openapi.OpenAPI) {
	api = i
}

func Register(config *Config, handles ...interface{}) {
	if config.Private {
		privateConfig[config.ID] = config
		privateHandles[config.ID] = handles
		return
	}
	if config.ID != "" {
		idConfig[config.ID] = config
		idHandles[config.ID] = handles
	}
	if config.Name != "" {
		nameConfig[config.Name] = config
		nameHandles[config.Name] = handles
	}
	for _, alias := range config.Alias {
		nameConfig[alias] = config
		nameHandles[alias] = handles
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

	config, ok := nameConfig[cmdName]
	if !ok {
		return
	}

	Run(&Context{
		Context: context.Background(),
		Api:     api,
		Data:    data,
		Msg:     msg,
		Cmd:     config,
		CmdName: cmdName,
		Args:    cmdArgs,
	})
	return
}

func GetPrivateConfig(id string) (*Config, bool) {
	config, ok := privateConfig[id]
	return config, ok
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
		h, ok := privateHandles[ctx.Cmd.ID]
		if !ok {
			err = errors.New(fmt.Sprintf("Cannot find %v command handle", ctx.Cmd.ID))
			return
		}
		handles = h
	} else {
		h, ok := idHandles[ctx.Cmd.ID]
		if !ok {
			err = errors.New(fmt.Sprintf("Cannot find %v command handle", ctx.Cmd.ID))
			return
		}
		handles = h
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
				invokeParams = append(invokeParams, reflect.New(paramType))
			} else {
				val, er := convArg(ctx.Args[j-1], paramType)
				if er != nil {
					continue handle
				}
				invokeParams = append(invokeParams, reflect.ValueOf(val))
			}
		}
		return handle, invokeParams, nil
	}
	msg := "⚠ 参数格式错误"
	if ctx.Cmd.Usage != "" {
		msg += "\n❓ 用法：" + ctx.Cmd.Usage
	}
	err = errors.New(msg)
	return
}
