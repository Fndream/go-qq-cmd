package cmd

// 在拦截器中：
// 如果返回的bool为true， 继续执行下一个拦截器
// 如果返回的bool为false，取消本次指令的执行，此时，如果同时返回的error不为nil，会回复错误消息

var globalBeforeHandle []func(ctx *Context) (bool, error)
var globalAfterHandle []func(ctx *Context) (bool, error)

var groupBeforeHandle map[string][]func(ctx *Context) (bool, error)
var groupAfterHandle map[string][]func(ctx *Context) (bool, error)

func SetGlobalBeforeHandle(f ...func(ctx *Context) (bool, error)) {
	globalBeforeHandle = f
}

func SetGlobalAfterHandle(f ...func(ctx *Context) (bool, error)) {
	globalAfterHandle = f
}

func SetGroupBeforeHandle(group string, f ...func(ctx *Context) (bool, error)) {
	if groupBeforeHandle == nil {
		groupBeforeHandle = make(map[string][]func(ctx *Context) (bool, error))
	}
	groupBeforeHandle[group] = f
}

func SetGroupAfterHandle(group string, f ...func(ctx *Context) (bool, error)) {
	if groupAfterHandle == nil {
		groupAfterHandle = make(map[string][]func(ctx *Context) (bool, error))
	}
	groupAfterHandle[group] = f
}
