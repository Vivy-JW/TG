using System;
using System.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000004 RID: 4
	public class SpecNodeTransform
	{
		// Token: 0x0600002B RID: 43 RVA: 0x0000265C File Offset: 0x0000085C
		internal static void TransformRequired(AbstractSpecNode spec, XmlElement formElement)
		{
			SpecFieldNode specFieldNode = spec as SpecFieldNode;
			if (specFieldNode == null || formElement == null)
			{
				return;
			}
			bool flag = string.Equals("y", specFieldNode.Req, StringComparison.CurrentCultureIgnoreCase);
			string attribute = formElement.GetAttribute("style");
			if (attribute == null)
			{
				return;
			}
			attribute.Split(new char[] { ' ' }).Distinct<string>().ToList<string>();
			switch (flag)
			{
			case false:
				formElement.IsRequired = false;
				return;
			case true:
				formElement.IsRequired = true;
				return;
			default:
				return;
			}
		}

		// Token: 0x0600002C RID: 44 RVA: 0x000026E0 File Offset: 0x000008E0
		internal static void TransformCanEdit(AbstractSpecNode spec, XmlElement formElement)
		{
			SpecFieldNode specFieldNode = spec as SpecFieldNode;
			if (specFieldNode == null || formElement == null)
			{
				return;
			}
			formElement.SetAttribute("noEntry", ("Y" == specFieldNode.CanEdit) ? "false" : "true");
		}

		// Token: 0x0600002D RID: 45 RVA: 0x00002724 File Offset: 0x00000924
		internal static void TransformTableColumn(AbstractSpecNode spec, XmlElement formElement)
		{
			SpecFieldNode specFieldNode = spec as SpecFieldNode;
			if (specFieldNode == null || formElement == null)
			{
				return;
			}
			string table = specFieldNode.Table;
			string column = specFieldNode.Column;
			if (formElement.GetAttribute("sqlTabName") != null && formElement.GetAttribute("sqlTabName") != table)
			{
				formElement.SetAttribute("sqlTabName", table);
			}
			if (formElement.GetAttribute("colName") != null && formElement.GetAttribute("colName") != column)
			{
				formElement.SetAttribute("colName", column);
			}
			if (!string.IsNullOrEmpty(column))
			{
				if (formElement.GetAttribute("format") != null)
				{
					string defaultAttributeValue = TableColumnHelper.GetDefaultAttributeValue(table, column, "format");
					if (!string.IsNullOrEmpty(defaultAttributeValue))
					{
						formElement.SetAttribute("format", defaultAttributeValue);
					}
				}
				if (formElement.GetAttribute("case") != null)
				{
					string defaultAttributeValue2 = TableColumnHelper.GetDefaultAttributeValue(table, column, "case");
					if (!string.IsNullOrEmpty(defaultAttributeValue2))
					{
						formElement.SetAttribute("case", defaultAttributeValue2);
					}
				}
			}
			SpecNodeTransform.TransformFieldType(formElement);
		}

		// Token: 0x0600002E RID: 46 RVA: 0x00002814 File Offset: 0x00000A14
		internal static void TransformFieldType(XmlElement formElement)
		{
			if (formElement == null)
			{
				return;
			}
			string attribute = formElement.GetAttribute("sqlTabName");
			string attribute2 = formElement.GetAttribute("colName");
			if (!string.IsNullOrEmpty(attribute2))
			{
				formElement.SetAttribute("fieldType", "TABLE_COLUMN");
				if (formElement.Parent != null && formElement.Parent.Name.StartsWith("s_browse"))
				{
					formElement.SetAttribute("fieldType", "COLUMN_LIKE");
				}
				if (!formElement.Name.Equals(string.Format("{0}.{1}", attribute, attribute2)))
				{
					formElement.SetAttribute("fieldType", "COLUMN_LIKE");
					return;
				}
			}
			else
			{
				formElement.SetAttribute("fieldType", "NON_DATABASE");
			}
		}

		// Token: 0x0600002F RID: 47 RVA: 0x000028C0 File Offset: 0x00000AC0
		internal static void TransformFormToSpecField(SpecFieldNode spec, XmlElement formElement)
		{
			if (spec == null || formElement == null)
			{
				return;
			}
			if (formElement.GetAttribute("fieldType") == "TABLE_COLUMN" || formElement.GetAttribute("fieldType") == "COLUMN_LIKE")
			{
				spec.SetAttribute("table", formElement.GetAttribute("sqlTabName"));
				spec.SetAttribute("column", formElement.GetAttribute("colName"));
			}
		}

		// Token: 0x06000030 RID: 48 RVA: 0x00002930 File Offset: 0x00000B30
		internal static void TransformProgRelNode(SpecProgRelNode specProgRelNode, XmlElement xmlElement)
		{
			if (specProgRelNode == null)
			{
				return;
			}
			string text = specProgRelNode.DependField;
			if (string.IsNullOrEmpty(text))
			{
				return;
			}
			if (text.Split(new char[] { '.' }).Count<string>() != 2)
			{
				return;
			}
			text = text.Split(new char[] { '.' })[1];
			string text2 = string.Format("lbl_{0}", text);
			if (xmlElement.GetAttribute("text") == text2)
			{
				return;
			}
			string text3 = SettingManager.Get().GetTzpManger(specProgRelNode.ProgramKey).SpecificationInfo.GetFieldLocalStringText(text2);
			if (string.IsNullOrEmpty(text3))
			{
				text3 = TableColumnHelper.GetColumnTextByFullName(text);
				SettingManager.Get().GetTzpManger(specProgRelNode.ProgramKey).SpecificationInfo.SetFieldLocalStringText(text2, text3);
			}
			xmlElement.SetAttribute("text", text2);
			xmlElement.LocalString = text3;
		}

		// Token: 0x06000031 RID: 49 RVA: 0x00002A00 File Offset: 0x00000C00
		internal static void TransformReferenceNode(SpecReferenceNode reference, XmlElement formElement)
		{
			if (reference == null)
			{
				return;
			}
			if (formElement.GetAttribute("title") != null && reference.RefRtn != "")
			{
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(formElement.Key).SpecificationInfo;
				string text = string.Format("lbl_{0}", reference.RefRtn);
				if (specificationInfo.GetFieldLocalStringText(text) == null)
				{
					string columnText = TableColumnHelper.GetColumnText(reference.RefTable, reference.RefRtn);
					if (columnText != null)
					{
						specificationInfo.SetFieldLocalStringText(text, columnText);
					}
				}
				formElement.SetAttribute("title", text);
			}
			if (reference.RefRtn != "")
			{
				formElement.SetAttribute("comment", string.Format("cmt_{0}", reference.RefRtn));
			}
		}

		// Token: 0x06000032 RID: 50 RVA: 0x00002AB8 File Offset: 0x00000CB8
		internal static void TransformMultilangNode(SpecMultiLangNode specMultiLangNode, XmlElement formElement)
		{
			string langTable = specMultiLangNode.LangTable;
			string langRTN = specMultiLangNode.LangRTN;
			if (formElement.GetAttribute("sqlTabName") != langTable)
			{
				formElement.SetAttribute("sqlTabName", langTable);
			}
			if (formElement.GetAttribute("colName") != langRTN)
			{
				formElement.SetAttribute("colName", langRTN);
			}
			SpecNodeTransform.TransformFieldType(formElement);
		}
	}
}
