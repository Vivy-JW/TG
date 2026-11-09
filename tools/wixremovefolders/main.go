// 给 heat.exe 生成的 .wxs 补上卸载时要删的目录登记。
//
// 为什么需要：这个 MSI 是**用户级**安装，目录都落在用户配置文件下（
// %LOCALAPPDATA%\Programs\TT）。MSI 的 ICE64 校验要求这类目录必须登记进
// RemoveFile 表，否则卸载后目录会原样留在磁盘上。heat 只生成组件与文件，
// 不生成 RemoveFolder，所以在构建流水线里补这一道。
//
// 规则（照 MSI 的约束来）：RemoveFolder 必须挂在某个 Component 里，且它指向的
// 目录要是该组件所在目录或它的上级。于是：
//
//	· 目录自己带组件（skill 目录就是这种）→ 往它的第一个组件里塞一个 RemoveFolder；
//	· 目录自己不带组件（skills/ 这种只有子目录的）→ 补一个只挂注册表键的组件，
//	  专门负责卸载时删掉这个空目录，并把它登记进 ComponentGroup；
//	· 只挂注册表键的组件用 Guid="*"：WiX 会按 keypath 推出稳定的 GUID，
//	  重打包装升级时不会认成另一个组件。
//
// 另一条（ICE38）：用户级安装里，组件的 KeyPath 必须是 HKCU 下的注册表键，不能用
// 文件 —— 用文件做 KeyPath 时，MSI 是按"文件版本"判定该组件装没装，这个判定是
// 全机范围的，和"按用户安装"的语义对不上。改成注册表键之后，每个用户各自的安装
// 状态是清楚的。
//
// 前身是 Python 脚本 tools/wix_removefolders.py，2026-10 随「tools 去 Python」改成
// Go。实现走"极简 DOM + 原位改写"：heat 输出没有文本内容与注释，元素树之外的信息
// 丢弃无碍；语义等价即可，candle 不在乎字节形态。
//
// 用法：go run ./tools/wixremovefolders <heat 生成的 .wxs>
// 就地改写。
package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

const wixNS = "http://schemas.microsoft.com/wix/2006/wi"

// attr 是一个 XML 属性。Go 的解码器把 xmlns 声明也放进 attrs
// （默认声明 Name = {Space:"", Local:"xmlns"}），序列化时原样带回。
type attr struct {
	space, local, value string
}

// node 是元素或字符数据。character 数据原样保留（含缩进空白），保证
// 未被触碰的部分尽量贴近 heat 的输出形态。
type node struct {
	elem   *element
	text   string
	isText bool
}

type element struct {
	space string // 解析出的命名空间 URI；等于 wixNS 的元素序列化时按本地名输出（继承根的默认 xmlns）
	local string
	attrs []attr
	kids  []node
}

func isWix(e *element) bool { return e.space == wixNS }

func (e *element) attr(local string) string {
	for _, a := range e.attrs {
		if a.local == local {
			return a.value
		}
	}
	return ""
}

func (e *element) setAttr(local, value string) {
	for i := range e.attrs {
		if e.attrs[i].local == local {
			e.attrs[i].value = value
			return
		}
	}
	e.attrs = append(e.attrs, attr{local: local, value: value})
}

func (e *element) removeAttr(local string) {
	for i := range e.attrs {
		if e.attrs[i].local == local {
			e.attrs = append(e.attrs[:i], e.attrs[i+1:]...)
			return
		}
	}
}

// childElems 返回直接子元素里 local 名匹配的那些（字符数据跳过）。
func (e *element) childElems(local string) []*element {
	var out []*element
	for i := range e.kids {
		if !e.kids[i].isText && e.kids[i].elem.local == local {
			out = append(out, e.kids[i].elem)
		}
	}
	return out
}

// firstChildElem 返回第一个匹配的直接子元素；没有则 nil。
func (e *element) firstChildElem(local string) *element {
	for i := range e.kids {
		if !e.kids[i].isText && e.kids[i].elem.local == local {
			return e.kids[i].elem
		}
	}
	return nil
}

func (e *element) appendChild(el *element) {
	e.kids = append(e.kids, node{elem: el})
}

// findAll 深度优先收集整棵树里 local 名匹配的元素。
func findAll(root *element, local string) []*element {
	var out []*element
	var walk func(e *element)
	walk = func(e *element) {
		if e.local == local {
			out = append(out, e)
		}
		for i := range e.kids {
			if !e.kids[i].isText {
				walk(e.kids[i].elem)
			}
		}
	}
	walk(root)
	return out
}

func parse(path string) (*element, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	var stack []*element
	var root *element
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			e := &element{space: t.Name.Space, local: t.Name.Local}
			for _, a := range t.Attr {
				e.attrs = append(e.attrs, attr{space: a.Name.Space, local: a.Name.Local, value: a.Value})
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.appendChild(e)
			} else if root == nil {
				root = e
			}
			stack = append(stack, e)
		case xml.CharData:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.kids = append(parent.kids, node{isText: true, text: string(t)})
			}
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		default:
			// ProcInst / Comment / Directive：heat 输出里没有，丢弃（与旧 ElementTree 行为一致）。
		}
	}
	if root == nil {
		return nil, fmt.Errorf("%s 里没有根元素", path)
	}
	return root, nil
}

func escapeText(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

func writeElem(e *element, b *strings.Builder) {
	b.WriteString("<" + e.local)
	for _, a := range e.attrs {
		name := a.local
		if a.space == "xmlns" {
			name = "xmlns:" + a.local
		}
		// 空格里的其他 URI 只会出现在本工具自己造不出、heat 也不产出的输入里；
		// 真遇到就按本地名输出，宁可交给 candle 报错也不悄悄改名。
		b.WriteString(" " + name + `="` + escapeText(a.value) + `"`)
	}
	if len(e.kids) == 0 {
		b.WriteString("/>")
		return
	}
	b.WriteString(">")
	for _, k := range e.kids {
		if k.isText {
			b.WriteString(escapeText(k.text))
		} else {
			writeElem(k.elem, b)
		}
	}
	b.WriteString("</" + e.local + ">")
}

// removeFolderEl 造一条「卸载时删掉 did 目录」的登记。
func removeFolderEl(did string) *element {
	return &element{space: wixNS, local: "RemoveFolder", attrs: []attr{
		{local: "Id", value: "rm_" + did},
		{local: "Directory", value: did},
		{local: "On", value: "uninstall"},
	}}
}

// keyPathRegistryEl 造一个 HKCU 注册表键 KeyPath（ICE38 要求的用户级组件判定）。
func keyPathRegistryEl(key, name string) *element {
	return &element{space: wixNS, local: "RegistryValue", attrs: []attr{
		{local: "Root", value: "HKCU"},
		{local: "Key", value: key},
		{local: "Name", value: name},
		{local: "Type", value: "integer"},
		{local: "Value", value: "1"},
		{local: "KeyPath", value: "yes"},
	}}
}

// patch 就地改写组件树，返回补出来的空目录组件数。
func patch(root *element) int {
	added := 0
	groups := findAll(root, "ComponentGroup")
	for _, d := range findAll(root, "Directory") {
		did := d.attr("Id")
		if did == "" {
			continue
		}
		comps := d.childElems("Component")
		if len(comps) > 0 {
			// 目录里有文件:在它的第一个组件里登记「卸载时删掉本目录」
			comps[0].appendChild(removeFolderEl(did))
			continue
		}
		// 只有子目录、自己没文件:补一个「空目录组件」,否则它会留在磁盘上。
		// 不写 Directory 属性:组件嵌在 <Directory> 里时该属性必须省略(否则 CNDL0062),
		// 所在目录由嵌套位置决定,也就是它自己。
		c := &element{space: wixNS, local: "Component", attrs: []attr{
			{local: "Id", value: "rmdir_" + did},
			{local: "Guid", value: "*"},
		}}
		c.appendChild(removeFolderEl(did))
		c.appendChild(keyPathRegistryEl(`Software\TT\dirs`, did))
		d.appendChild(c)
		for _, g := range groups {
			g.appendChild(&element{space: wixNS, local: "ComponentRef", attrs: []attr{
				{local: "Id", value: "rmdir_" + did},
			}})
		}
		added++
	}

	// ICE38:用户级安装里,组件的 KeyPath 必须是 HKCU 下的注册表键,不能用文件 ——
	// 用文件做 KeyPath 时,MSI 是按"文件版本"判定该组件装没装,这个判定是全机范围的,
	// 和"按用户安装"的语义对不上。改成注册表键之后,每个用户各自的安装状态是清楚的。
	for _, c := range findAll(root, "Component") {
		f := c.firstChildElem("File")
		if f == nil || f.attr("KeyPath") != "yes" {
			continue
		}
		cid := c.attr("Id")
		f.removeAttr("KeyPath")
		c.appendChild(keyPathRegistryEl(`Software\TT\components`, cid))
	}
	return added
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("用法: go run ./tools/wixremovefolders <heat 生成的 .wxs>")
	}
	path := os.Args[1]
	root, err := parse(path)
	if err != nil {
		return err
	}
	added := patch(root)

	var b strings.Builder
	b.WriteString(xml.Header)
	writeElem(root, &b)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("wix_removefolders: %s (补了 %d 个空目录组件)\n", path, added)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
