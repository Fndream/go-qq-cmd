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

type Context struct {
	context.Context
	Api     openapi.OpenAPI // api
	Data    *dto.Message    // 事件数据
	Direct  bool            // 是否是私聊事件
	Msg     string          // 消息内容
	Cmd     *Command        // 指令
	CmdName string          // 指令名
	Args    []interface{}   // 参数
}

type Command struct {
	*Config
	Handles []interface{}
}

type Config struct {
	ID             string      // ID
	Group          []string    // 分组
	Name           string      // 名称
	Alias          []string    // 别名
	Usage          string      // 用法
	Emoji          string      // emoji图标
	Description    string      // 描述
	Permission     *Permission // 权限
	DisableChannel bool        // 是否在子频道禁用
	DisableDirect  bool        // 是否在私信禁用
	Private        bool        // 是否为内部指令
	Async          bool        // 是否可异步执行
}

var idMap = make(map[string]*Command)
var nameMap = make(map[string]*Command)
var privateMap = make(map[string]*Command)

func Register(config *Config, handles ...interface{}) {
	cmd := Command{
		Config:  config,
		Handles: handles,
	}
	if cmd.Permission == nil {
		cmd.Permission = Member
	}
	if config.Private {
		privateMap[config.ID] = &cmd
		return
	}
	if config.ID != "" {
		_, ok := idMap[config.ID]
		if ok {
			panic("重复注册的指令ID: " + config.ID + "在 Name: " + config.Name + " 中")
		}
		idMap[config.ID] = &cmd
	}
	if config.Name != "" {
		_, ok := nameMap[config.Name]
		if ok {
			panic("重复注册的指令名称: " + config.Name + "在 ID: " + config.ID + "中")
		}
		nameMap[config.Name] = &cmd
	}
	for _, alias := range config.Alias {
		_, ok := nameMap[alias]
		if ok {
			panic("重复注册的指令别名: " + alias + "在 ID: " + config.ID + " Name: " + config.Name + " 中")
		}
		nameMap[alias] = &cmd
	}
}

func Process(data *dto.Message) {
	msg := strings.Trim(data.Content, " ")
	msg = strings.Trim(data.Content, "\n")
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

	var args []interface{}
	for _, s := range cmdArgs {
		args = append(args, s)
	}

	ctx := &Context{
		Context: context.Background(),
		Api:     api,
		Data:    data,
		Direct:  data.DirectMessage,
		Msg:     msg,
		CmdName: cmdName,
		Args:    args,
	}

	cmd, cmdOk := nameMap[cmdName]
	if cmdOk {
		ctx.Cmd = cmd
		if !canRun(ctx) {
			return
		}
	}

	ds, dlOk := userDialogs.Load(ctx.Data.Author.ID)
	if dlOk {
		dl := ds.(*DialogStack).Last()
		dlctx := dl.GetCtx()
		if ctx.Data.ChannelID == dlctx.Data.ChannelID {
			if cmdOk {
				SendReplyS(ctx, "💬 还有未回复的对话\n"+dl.GetMainMsgView().Msg)
				return
			}
			dl.GetChannel() <- ctx
			return
		}
		if cmdOk {
			SendReplyS(ctx, "💬 还有未回复的对话")
		}
		return
	}

	if cmdOk {
		Run(ctx)
	}
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
	if !canRun(ctx) {
		return
	}

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

func GetPrivateCommand(id string) (*Command, bool) {
	cmd, ok := privateMap[id]
	return cmd, ok
}

func canRun(ctx *Context) bool {
	// 指令环境
	if (!ctx.Direct && ctx.Cmd.DisableChannel) || (ctx.Direct && ctx.Cmd.DisableDirect) {
		return false
	}
	if !ctx.Data.DirectMessage {
		// 子频道
		// 频道黑名单
		if inSlice(ctx.Data.ChannelID, bwcfg.BlackChannels[ctx.Data.GuildID]) {
			return false
		}
		// 频道白名单
		if len(bwcfg.WhiteChannels) > 0 && !inSlice(ctx.Data.ChannelID, bwcfg.WhiteChannels[ctx.Data.GuildID]) {
			return false
		}
	}
	// 用户黑名单
	for _, user := range bwcfg.BlackUsers {
		if user.Id == ctx.Data.Author.ID {
			return false
		}
	}
	// 权限
	uPermission := parsePermission(ctx.Data.Member.Roles)
	if uPermission.Level < ctx.Cmd.Permission.Level {
		SendReplyS(ctx, "🔑 权限不足")
		return false
	}
	return true
}

func parsePermission(roles []string) *Permission {
	if roles == nil {
		return Member
	}
	switch roles[0] {
	case "1":
		return Member
	case "2":
		return Admin
	case "4":
		return Owner
	case "5":
		return ChannelAdmin
	}
	return Member
}

func inSlice(s string, slice []string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
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

		paramCount := handleType.NumIn()
		for j := 1; j < paramCount; j++ {
			paramType := handleType.In(j)
			if j-1 >= len(ctx.Args) || (ctx.Args[j-1] == "_" || ctx.Args[j-1] == ".") {
				if paramType.Kind() == reflect.Pointer {
					invokeParams = append(invokeParams, reflect.New(paramType).Elem())
					continue
				}
				if j == paramCount-1 && handleType.IsVariadic() {
					continue
				}
				continue handle
			}

			if paramType.Kind() == reflect.Pointer {
				if reflect.TypeOf(ctx.Args[j-1]).Kind() == reflect.Pointer {
					if reflect.TypeOf(ctx.Args[j-1]).Elem().Kind() == paramType.Elem().Kind() {
						invokeParams = append(invokeParams, reflect.ValueOf(ctx.Args[j-1]))
						continue
					}
				} else if reflect.TypeOf(ctx.Args[j-1]).Kind() == paramType.Elem().Kind() {
					val := reflect.New(paramType.Elem())
					val.Elem().Set(reflect.ValueOf(ctx.Args[j-1]))
					invokeParams = append(invokeParams, val)
					continue
				}
			} else if reflect.TypeOf(ctx.Args[j-1]).Kind() == paramType.Kind() {
				invokeParams = append(invokeParams, reflect.ValueOf(ctx.Args[j-1]))
				continue
			}

			if j == paramCount-1 && handleType.IsVariadic() {
				for _, arg := range ctx.Args[j-1:] {
					v, e := convArg(arg.(string), paramType.Elem())
					if e != nil {
						continue handle
					}
					invokeParams = append(invokeParams, reflect.ValueOf(v))
				}
				continue
			}

			if paramType.Kind() == reflect.Pointer {
				v, e := convArg(fmt.Sprintf("%v", ctx.Args[j-1]), paramType.Elem())
				if e != nil {
					continue handle
				}
				val := reflect.New(paramType.Elem())
				val.Elem().Set(reflect.ValueOf(v))
				invokeParams = append(invokeParams, val)
			} else {
				v, e := convArg(fmt.Sprintf("%v", ctx.Args[j-1]), paramType)
				if e != nil {
					continue handle
				}
				invokeParams = append(invokeParams, reflect.ValueOf(v))
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

func init() {
	loadConfig()
	Register(&Config{
		ID:         "fetch-channel-Id",
		Group:      []string{"builtin-admin"},
		Name:       "cid",
		Emoji:      "🆔",
		Permission: Admin,
		Async:      true,
	}, fetchChannelId)
	Register(&Config{
		ID:         "add-black-users",
		Group:      []string{"builtin-admin"},
		Name:       "ban",
		Emoji:      "⚫",
		Permission: Admin,
		Async:      true,
	}, addBlackUsers)
	Register(&Config{
		ID:         "remove-ban-users",
		Group:      []string{"builtin-admin"},
		Name:       "rmban",
		Emoji:      "⚫",
		Permission: Admin,
		Async:      true,
	}, removeBlackUsers)
	Register(&Config{
		ID:         "add-black-channels",
		Group:      []string{"builtin-admin"},
		Name:       "banc",
		Emoji:      "⚫",
		Permission: Admin,
		Async:      true,
	}, addBlackChannels)
	Register(&Config{
		ID:         "remove-ban-channels",
		Group:      []string{"builtin-admin"},
		Name:       "rmbanc",
		Emoji:      "⚫",
		Permission: Admin,
		Async:      true,
	}, removeBanChannels)
	Register(&Config{
		ID:         "add-white-channels",
		Group:      []string{"builtin-admin"},
		Name:       "wc",
		Emoji:      "⚪",
		Permission: Admin,
		Async:      true,
	}, addWhiteChannels)
	Register(&Config{
		ID:         "remove-white-channels",
		Group:      []string{"builtin-admin"},
		Name:       "rmwc",
		Emoji:      "⚪",
		Permission: Admin,
		Async:      true,
	}, removeWhiteChannels)
	Register(&Config{
		ID:         "query-black-users",
		Group:      []string{"builtin-admin"},
		Name:       "qban",
		Emoji:      "⚫",
		Permission: Admin,
		Async:      true,
	}, queryBlackUsers)
	Register(&Config{
		ID:         "query-black-channels",
		Group:      []string{"builtin-admin"},
		Name:       "qbc",
		Emoji:      "⚫",
		Permission: Admin,
		Async:      true,
	}, queryBlackChannels)
	Register(&Config{
		ID:         "query-white-channels",
		Group:      []string{"builtin-admin"},
		Name:       "qwc",
		Emoji:      "⚪",
		Permission: Admin,
		Async:      true,
	}, queryWhiteChannels)
}
