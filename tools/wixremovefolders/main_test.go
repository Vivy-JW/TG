package main

import (
	"encoding/xml"
	"strings"
	"testing"
)

// 夹具模拟 heat.exe -sfrag -srd -dr INSTALLFOLDER 的输出形态，覆盖三条规则：
//
//	· skills.ttzs —— 目录自己带组件 → 往第一个组件里塞 RemoveFolder；
//	· skills     —— 只有子目录、自己没组件 → 补一个只挂注册表键的空目录组件，
//	                并在 ComponentGroup 里登记 ComponentRef；
//	· cmp1      —— File 做 KeyPath 的组件 → KeyPath 挪到 HKCU 注册表键（ICE38）。
const fixture = `<?xml version="1.0" encoding="utf-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
  <Fragment>
    <ComponentGroup Id="StagedFiles">
      <Component Id="cmp1" Directory="INSTALLFOLDER" Guid="*">
        <File Id="file1" Source="$(var.SourceDir)\tt.exe" KeyPath="yes" />
      </Component>
    </ComponentGroup>
    <DirectoryRef Id="INSTALLFOLDER">
      <Directory Id="skills" Name="skills">
        <Directory Id="skills.ttzs" Name="tt-dev-tzs">
          <Component Id="cmpSkill" Guid="*">
            <File Id="fileSkill" Source="$(var.SourceDir)\skills\tt-dev-tzs\SKILL.md" KeyPath="yes" />
          </Component>
        </Directory>
      </Directory>
    </DirectoryRef>
  </Fragment>
</Wix>
`

func parseFixture(t *testing.T) *element {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(fixture))
	var stack []*element
	var root *element
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			e := &element{space: t.Name.Space, local: t.Name.Local}
			for _, a := range t.Attr {
				e.attrs = append(e.attrs, attr{space: a.Name.Space, local: a.Name.Local, value: a.Value})
			}
			if len(stack) > 0 {
				stack[len(stack)-1].appendChild(e)
			} else {
				root = e
			}
			stack = append(stack, e)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
	if root == nil {
		t.Fatal("夹具解析不出根元素")
	}
	return root
}

func TestPatchInjectsUninstallSemantics(t *testing.T) {
	root := parseFixture(t)
	added := patch(root)
	if added != 1 {
		t.Fatalf("空目录组件数 = %d，要 1（只有 skills/ 一个）", added)
	}

	// 规则一：带组件的目录，RemoveFolder 进它的第一个组件。
	skillDir := findDir(t, root, "skills.ttzs")
	cmpSkill := skillDir.firstChildElem("Component")
	if got := cmpSkill.firstChildElem("RemoveFolder"); got == nil || got.attr("Id") != "rm_skills.ttzs" ||
		got.attr("Directory") != "skills.ttzs" || got.attr("On") != "uninstall" {
		t.Fatalf("skills.ttzs 的组件里没有正确的 RemoveFolder：%+v", got)
	}

	// 规则二：只有子目录的目录，补注册表键组件 + ComponentRef 登记。
	skillsDir := findDir(t, root, "skills")
	rmdir := skillsDir.firstChildElem("Component")
	if rmdir == nil || rmdir.attr("Id") != "rmdir_skills" || rmdir.attr("Guid") != "*" {
		t.Fatalf("skills/ 下没有补出 rmdir_skills 组件：%+v", rmdir)
	}
	if rmdir.attr("Directory") != "" {
		// CNDL0062：嵌套在 <Directory> 里的组件必须省略 Directory 属性。
		t.Fatalf("嵌套组件带了 Directory 属性（CNDL0062 会红）：%q", rmdir.attr("Directory"))
	}
	rm := rmdir.firstChildElem("RemoveFolder")
	if rm == nil || rm.attr("Directory") != "skills" {
		t.Fatalf("rmdir_skills 里没有指向 skills/ 的 RemoveFolder")
	}
	reg := rmdir.firstChildElem("RegistryValue")
	if reg == nil || reg.attr("Root") != "HKCU" || reg.attr("Key") != `Software\TT\dirs` ||
		reg.attr("Name") != "skills" || reg.attr("KeyPath") != "yes" {
		t.Fatalf("rmdir_skills 的 KeyPath 注册表键不对：%+v", reg)
	}
	group := findAll(root, "ComponentGroup")[0]
	found := false
	for _, ref := range group.childElems("ComponentRef") {
		if ref.attr("Id") == "rmdir_skills" {
			found = true
		}
	}
	if !found {
		t.Fatal("ComponentGroup 里没有 rmdir_skills 的 ComponentRef")
	}

	// 规则三：File KeyPath 挪到 HKCU 注册表键。
	cmp1 := findComponent(t, root, "cmp1")
	if f := cmp1.firstChildElem("File"); f.attr("KeyPath") != "" {
		t.Fatalf("cmp1 的 File 还留着 KeyPath（ICE38 会红）")
	}
	reg1 := cmp1.firstChildElem("RegistryValue")
	if reg1 == nil || reg1.attr("Key") != `Software\TT\components` || reg1.attr("Name") != "cmp1" ||
		reg1.attr("KeyPath") != "yes" {
		t.Fatalf("cmp1 的 KeyPath 注册表键不对：%+v", reg1)
	}
}

func TestRoundTripPreservesUnpatchedParts(t *testing.T) {
	root := parseFixture(t)
	patch(root)

	var b strings.Builder
	b.WriteString(xml.Header)
	writeElem(root, &b)
	out := b.String()

	for _, want := range []string{
		`<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">`, // 根的默认命名空间必须原样带回
		`Id="cmp1"`, `Id="cmpSkill"`, `$(var.SourceDir)\tt.exe`, // 未触碰的属性/文本原样保留
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("序列化输出里找不到 %q：\n%s", want, out)
		}
	}
	if strings.Contains(out, "ns0:") {
		t.Fatalf("序列化输出带出了 ns0: 前缀（默认命名空间丢了）：\n%s", out)
	}
}

func findDir(t *testing.T, root *element, id string) *element {
	t.Helper()
	for _, d := range findAll(root, "Directory") {
		if d.attr("Id") == id {
			return d
		}
	}
	t.Fatalf("找不到 Directory %q", id)
	return nil
}

func findComponent(t *testing.T, root *element, id string) *element {
	t.Helper()
	for _, c := range findAll(root, "Component") {
		if c.attr("Id") == id {
			return c
		}
	}
	t.Fatalf("找不到 Component %q", id)
	return nil
}
