package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

func convArg(arg string, target reflect.Type) (value interface{}, err error) {
	// 参数转换不能崩，不然无法返回给用户"参数格式错误"
	defer func() {
		if er := recover(); er != nil {
			err = errors.New(er.(string))
		}
	}()
	switch target.Kind() {
	case reflect.Int:
		return strconv.Atoi(arg)
	case reflect.String, reflect.Interface:
		return arg, nil
	case reflect.Float32:
		return strconv.ParseFloat(arg, 32)
	case reflect.Float64:
		return strconv.ParseFloat(arg, 64)
	case reflect.Bool:
		return strconv.ParseBool(arg)
	case reflect.Slice:
		i := 0
		return convSlice(arg, target, &i)
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported conver type: %v", target.Name()))
	}
}

func convSlice(arg string, target reflect.Type, i *int) (interface{}, error) {
	slice := reflect.MakeSlice(target, 0, 16)
	e := ""
	ch := []rune(arg)
	for ; *i < len(ch); *i++ {
		c := ch[*i]

		switch c {
		case '[': // 开始切片
			*i++
			sli, er := convSlice(arg, target.Elem(), i)
			if er != nil {
				return nil, er
			}
			slice = reflect.Append(slice, reflect.ValueOf(sli))
		case ',': // 元素结束
			if len(e) != 0 {
				arg, err := convArg(e, target.Elem())
				if err != nil {
					return nil, err
				}
				slice = reflect.Append(slice, reflect.ValueOf(arg))
				e = ""
			}
		case ']': // 切片结束
			*i++
			if len(e) != 0 {
				arg, err := convArg(e, target.Elem())
				if err != nil {
					return nil, err
				}
				slice = reflect.Append(slice, reflect.ValueOf(arg))
				e = ""
			}
			return slice.Interface(), nil
		case '\\': // 转义
			if *i < len(ch)-1 {
				*i++
				str, er := strconv.Unquote(`"` + string(c) + string(ch[*i]) + `"`)
				if er != nil {
					e += string(ch[*i])
					break
				}
				e += str
				break
			}
			e += string(c)
		default:
			e += string(c)
		}
	}
	if len(e) != 0 {
		arg, err := convArg(e, target.Elem())
		if err != nil {
			return nil, err
		}
		slice = reflect.Append(slice, reflect.ValueOf(arg))
	}
	return slice.Interface(), nil
}

func parseMessageArgs(msg string) (args []string) {
	if len(msg) <= 0 {
		return []string{}
	}
	args = make([]string, 0, 16)
	e := ""
	in := false
	ch := []rune(msg)
	for i := 0; i < len(ch); i++ {
		c := ch[i]
		switch c {
		case '"': // 在引号内
			in = !in
		case ' ': // 分词
			if in {
				e += string(c)
				break
			}
			if len(e) > 0 {
				args = append(args, e)
				e = ""
			}
		case '\\': // 转义
			if i < len(ch)-1 {
				i++
				str, err := strconv.Unquote(`"` + string(c) + string(ch[i]) + `"`)
				if err != nil {
					e += string(ch[i])
					break
				}
				e += str
				break
			}
			e += string(c)
		default:
			e += string(c)
		}
	}
	if len(e) > 0 {
		args = append(args, e)
	}
	return
}
