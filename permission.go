package cmd

type Permission struct {
	Name  string // 权限名
	Level int    // 权限等级
}

var (
	Member       = &Permission{Name: "成员", Level: 0}    // 普通成员
	ChannelAdmin = &Permission{Name: "频道管理员", Level: 1} // 子频道管理员
	Admin        = &Permission{Name: "超级管理员", Level: 2} // 超级管理员
	Owner        = &Permission{Name: "频道主", Level: 3}   // 频道主
)
