package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
	"reflect"
	"strings"
	"sync"
	"unicode/utf8"
)

const spaceCharSet = " \u00A0"

type Result struct {
	Msg   string
	NotAt bool
}

type Context struct {
	context.Context
	Api     *openapi.OpenAPI     // api
	Data    *dto.WSATMessageData // 事件数据
	Msg     string               // 消息内容
	Cmd     *Config              // 指令信息
	CmdName string               // 指令名
	Args    []string             // 参数
}

type Config struct {
	Private     bool   // 是否内部指令
	ID          string // ID
	Name        string
	Alias       []string
	Usage       string
	Emoji       string
	Description string
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

func Process(data *dto.WSATMessageData) error {
	msg := strings.Trim(data.Content, spaceCharSet)
	if msg == "" {
		return nil
	}

	msgArgs := parseMessageArgs(msg)
	cmdName := msgArgs[1][1:]
	cmdArgs := msgArgs[2:]
	config, ok := nameConfig[cmdName]
	if !ok {
		return nil
	}

	ctx := &Context{
		Context: context.Background(),
		Api:     api,
		Data:    data,
		Msg:     msg,
		Cmd:     config,
		CmdName: cmdName,
		Args:    cmdArgs,
	}

	result, err := Run(ctx, false)
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
		})
		return err
	}

	if result != nil {
		if result.NotAt {
			SendReplyNotAt(ctx, &dto.MessageToCreate{
				Content: result.Msg,
			})
		} else {
			SendReply(ctx, &dto.MessageToCreate{
				Content: result.Msg,
			})
		}
	}

	return nil
}

func GetPrivateConfig(id string) (*Config, bool) {
	config, ok := privateConfig[id]
	return config, ok
}

func Run(ctx *Context, private bool) (result *Result, err error) {
	var mutex *sync.Mutex
	defer func() {
		if mutex != nil {
			unLock(mutex)
		}
		if er := recover(); er != nil {
			err = er.(error)
		}
	}()

	var handles []interface{}
	if private {
		h, ok := privateHandles[ctx.Cmd.ID]
		if !ok {
			return nil, errors.New(fmt.Sprintf("Cannot find %v command handle", ctx.Cmd.ID))
		}
		handles = h
	} else {
		h, ok := idHandles[ctx.Cmd.ID]
		if !ok {
			return nil, errors.New(fmt.Sprintf("Cannot find %v command handle", ctx.Cmd.ID))
		}
		handles = h
	}

	// 遍历处理函数
handle:
	for _, handle := range handles {
		// 得到处理器的类型
		handleType := reflect.TypeOf(handle)

		// 创建一个容量与处理器参数数量相等的切片，用来传递参数
		invokeParams := make([]reflect.Value, 0, handleType.NumIn())
		invokeParams = append(invokeParams, reflect.ValueOf(ctx))

		// 遍历参数
		for j := 1; j < handleType.NumIn(); j++ {
			// 得到参数类型
			paramType := handleType.In(j)

			// 判断参数是否需要留空 (参数不足或为留空占位符)
			if j-1 >= len(ctx.Args) || ctx.Args[j-1] == "_" {
				// 如果参数类型是指针，代表留空，传入nil，否则放弃该函数
				if paramType.Kind() != reflect.Pointer {
					continue handle
				}
				invokeParams = append(invokeParams, reflect.New(paramType))
			} else {
				// 转换参数类型并加入参数
				val, er := convArg(ctx.Args[j-1], paramType)
				if er != nil {
					continue handle
				}
				invokeParams = append(invokeParams, reflect.ValueOf(val))
			}
		}

		mutex = lock(ctx.Data.Author.ID)
		r := reflect.ValueOf(handle).Call(invokeParams)
		unLock(mutex)
		mutex = nil

		result = r[0].Interface().(*Result)
		e := r[1].Interface()
		if e != nil {
			err = e.(error)
		}
		return
	}
	msg := "⚠ 参数格式错误"
	if ctx.Cmd.Usage != "" {
		msg += "\n❓ 用法：" + ctx.Cmd.Usage
	}
	result = &Result{Msg: msg}
	return
}
