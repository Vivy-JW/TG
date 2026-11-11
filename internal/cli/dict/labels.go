package dict

import "fmt"

// custLabel 把 T100 字典里的客制旗标翻成中文（s=标准, c=客制）。
// scc / win / param / msg 几个命令共用。
func custLabel(code string) string {
	switch code {
	case "s":
		return "标准"
	case "c":
		return "客制"
	default:
		return code
	}
}

// typeWithCode 在中文标签后面补上原始代码，便于对照 T100 画面与 SQL。
// 代码为空或与标签相同时不重复显示。
func typeWithCode(label, code string) string {
	if code == "" || code == label {
		return label
	}
	return fmt.Sprintf("%s (%s)", label, code)
}

// checkDateTypeLabel 把日期型态码翻成中文（1=年月日, 2=年月日时分秒, 3=带毫秒）。
func checkDateTypeLabel(code string) string {
	switch code {
	case "1":
		return "年月日"
	case "2":
		return "年月日时分秒"
	case "3":
		return "年月日时分秒毫秒"
	default:
		return "字符串"
	}
}

// specWidgetLabel 把字段规格里的控件代码翻成 Genero 控件名（05=Edit、04=DateEdit…）。
func specWidgetLabel(code string) string {
	labels := map[string]string{
		"01": "ButtonEdit", "02": "CheckBox", "03": "ComboBox", "04": "DateEdit",
		"05": "Edit", "06": "FFImage", "07": "FFLabel", "08": "ProgressBar",
		"09": "RadioGroup", "10": "Slider", "11": "SpinEdit", "12": "TextEdit",
		"13": "TimeEdit", "14": "Field", "15": "WebComponent", "34": "DateTimeEdit",
	}
	label, ok := labels[code]
	if !ok {
		return code
	}
	return fmt.Sprintf("%s (%s)", label, code)
}
