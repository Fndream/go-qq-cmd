package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/tencent-connect/botgo/log"
	"github.com/tencent-connect/botgo/openapi"
	"os"
	"qqbot/fileutil"
	"regexp"
)

type bwConfig struct {
	BlackUsers    []*user             `json:"blackUsers,omitempty"`    // 用户黑名单
	BlackChannels map[string][]string `json:"blackChannels,omitempty"` // 子频道黑名单
	WhiteChannels map[string][]string `json:"whiteChannels,omitempty"` // 子频道白名单
}

type user struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

var api openapi.OpenAPI
var bwcfg = &bwConfig{
	BlackUsers:    []*user{},
	BlackChannels: map[string][]string{},
	WhiteChannels: map[string][]string{},
}

func SetApi(a openapi.OpenAPI) {
	api = a
}

func AddBlackUsers(us ...*user) {
	bwcfg.BlackUsers = append(bwcfg.BlackUsers, us...)
}

func RemoveBlackUsers(users ...*user) {
	var r []*user
	for _, v := range users {
		for _, v2 := range bwcfg.BlackUsers {
			if v.Id != v2.Id {
				r = append(r, v2)
			}
		}
	}
	if len(r) > 0 {
		bwcfg.BlackUsers = r
	}
}

func AddBlackChannels(gid string, cid ...string) {
	bwcfg.BlackChannels[gid] = append(bwcfg.BlackChannels[gid], cid...)
}

func RemoveBlackChannels(gid string, cid ...string) {
	var r []string
	for _, v := range cid {
		for _, v2 := range bwcfg.BlackChannels[gid] {
			if v != v2 {
				r = append(r, v2)
			}
		}
	}
	if len(r) > 0 {
		bwcfg.BlackChannels[gid] = r
	}
}

func AddWhiteChannels(gid string, cid ...string) {
	bwcfg.WhiteChannels[gid] = append(bwcfg.WhiteChannels[gid], cid...)
}

func RemoveWhiteChannels(gid string, cid ...string) {
	var r []string
	for _, v := range cid {
		for _, v2 := range bwcfg.WhiteChannels[gid] {
			if v != v2 {
				r = append(r, v2)
			}
		}
	}
	if len(r) > 0 {
		bwcfg.WhiteChannels[gid] = r
	}
}

func fetchChannelId(ctx *Context) (res *MsgView, err error) {
	res = &MsgView{Msg: "🆔 " + ctx.Data.ChannelID}
	return
}

func queryBlackUsers(ctx *Context) (res *MsgView, err error) {
	msg := "⚫ 用户黑名单：\n"
	for i, user := range bwcfg.BlackUsers {
		msg += "🆔 [" + user.Name + "]"
		if i%2 == 1 {
			msg += " "
		} else {
			msg += "\n"
		}
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func queryBlackChannels(ctx *Context) (res *MsgView, err error) {
	msg := "⚫ 子频道黑名单：\n"
	i := 1
	for _, cid := range bwcfg.BlackChannels[ctx.Data.GuildID] {
		msg += "🆔 " + cid
		if i%2 == 1 {
			msg += " "
		} else {
			msg += "\n"
		}
		i++
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func queryWhiteChannels(ctx *Context) (res *MsgView, err error) {
	msg := "⚪ 子频道白名单：\n"
	i := 1
	for _, cid := range bwcfg.WhiteChannels[ctx.Data.GuildID] {
		msg += "🆔 " + cid
		if i%2 == 1 {
			msg += " "
		} else {
			msg += "\n"
		}
		i++
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func addBlackUsers(ctx *Context) (res *MsgView, err error) {
	mentions := ctx.Data.Mentions
	var users []*user
	for _, mention := range mentions {
		if !mention.Bot {
			users = append(users, &user{Id: mention.ID, Name: mention.Username})
		}
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("⚫ 请至少at一个有效用户")
	}
	AddBlackUsers(users...)
	saveConfig()
	msg := "⚫ 已将下列用户添加到黑名单：\n"
	for _, user := range users {
		msg += "🆔 " + user.Name + "\n"
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func removeBlackUsers(ctx *Context) (res *MsgView, err error) {
	mentions := ctx.Data.Mentions
	var users []*user
	for _, mention := range mentions {
		if !mention.Bot {
			users = append(users, &user{Id: mention.ID, Name: mention.Username})
		}
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("⚫ 请至少at一个有效用户")
	}
	RemoveBlackUsers(users...)
	saveConfig()
	msg := "⚫ 已将下列用户从黑名单移除：\n"
	for _, user := range users {
		msg += "🆔 " + user.Name + "\n"
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func addBlackChannels(ctx *Context, ids []string) (res *MsgView, err error) {
	var cids []string
	for _, cid := range ids {
		if regexp.MustCompile(`^\d{5,}$`).MatchString(cid) {
			cids = append(cids, cid)
		}
	}
	if len(cids) == 0 {
		return nil, fmt.Errorf("⚫ 请至少输入一个有效的子频道ID")
	}
	AddBlackChannels(ctx.Data.GuildID, cids...)
	saveConfig()
	msg := "⚫ 已将下列子频道添加到黑名单：\n"
	for _, cid := range cids {
		msg += "🆔 " + cid + "\n"
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func removeBanChannels(ctx *Context, ids []string) (res *MsgView, err error) {
	var cids []string
	for _, cid := range ids {
		if regexp.MustCompile(`^\d{5,}$`).MatchString(cid) {
			cids = append(cids, cid)
		}
	}
	if len(cids) == 0 {
		return nil, fmt.Errorf("⚫ 请至少输入一个有效的子频道ID")
	}
	RemoveBlackChannels(ctx.Data.GuildID, cids...)
	saveConfig()
	msg := "⚫ 已将下列子频道从黑名单移除：\n"
	for _, cid := range cids {
		msg += "🆔 " + cid + "\n"
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func addWhiteChannels(ctx *Context, ids []string) (res *MsgView, err error) {
	var cids []string
	for _, cid := range ids {
		if regexp.MustCompile(`^\d{5,}$`).MatchString(cid) {
			cids = append(cids, cid)
		}
	}
	if len(cids) == 0 {
		return nil, fmt.Errorf("⚪ 请至少输入一个有效的子频道ID")
	}
	AddWhiteChannels(ctx.Data.GuildID, cids...)
	saveConfig()
	msg := "⚪ 已将下列子频道添加到白名单：\n"
	for _, cid := range cids {
		msg += "🆔 " + cid + "\n"
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func removeWhiteChannels(ctx *Context, ids []string) (res *MsgView, err error) {
	var cids []string
	for _, cid := range ids {
		if regexp.MustCompile(`^\d{5,}$`).MatchString(cid) {
			cids = append(cids, cid)
		}
	}
	if len(cids) == 0 {
		return nil, fmt.Errorf("⚪ 请至少输入一个有效的子频道ID")
	}
	RemoveWhiteChannels(ctx.Data.GuildID, cids...)
	saveConfig()
	msg := "⚪ 已将下列子频道从白名单移除：\n"
	for _, cid := range cids {
		msg += "🆔 " + cid + "\n"
	}
	msg = msg[:len(msg)-1]
	res = &MsgView{Msg: msg}
	return
}

func loadConfig() {
	path, _ := fileutil.GetPath("config/cmd-global.json")
	err := fileutil.CreateFileIfNotExistDefault(path, "{}")
	if err != nil {
		log.Errorf("创建配置文件失败: %v", err)
		return
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("加载配置文件失败: %v", err)
		return
	}
	if len(bytes) == 0 {
		return
	}
	err = json.Unmarshal(bytes, &bwcfg)
	if err != nil {
		log.Errorf("解析配置文件失败: %v", err)
		return
	}
}

func saveConfig() {
	data, _ := json.Marshal(bwcfg)
	path, _ := fileutil.GetPath("config/cmd-global.json")
	err := fileutil.CreateFileIfNotExistDefault(path, "{}")
	if err != nil {
		log.Errorf("创建配置文件失败: %v", err)
		return
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Errorf("保存配置文件失败: %v", err)
		return
	}
}
