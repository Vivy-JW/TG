using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000016 RID: 22
	public class ScreenRecordManager
	{
		// Token: 0x060000A9 RID: 169 RVA: 0x0000451A File Offset: 0x0000271A
		public ScreenRecordManager(XElement formSource)
		{
			this.Records = ((formSource != null) ? formSource.Descendants("Record").ToList<XElement>() : new List<XElement>());
		}

		// Token: 0x060000AA RID: 170 RVA: 0x000046D8 File Offset: 0x000028D8
		public IEnumerable<XElement> GetRecords()
		{
			foreach (XElement e in this.Records)
			{
				yield return e;
			}
			yield break;
		}

		// Token: 0x060000AB RID: 171 RVA: 0x000046F8 File Offset: 0x000028F8
		public void AddRecordField(XmlElement xmlElement)
		{
			if (xmlElement.Parent != null && (xmlElement.Parent.Type == ComponentType.Table || xmlElement.Parent.Type == ComponentType.Tree || xmlElement.Parent.Type == ComponentType.ScrollGrid))
			{
				this.AddRecordFieldTo(xmlElement, xmlElement.Parent.Name);
			}
			else if (xmlElement.Type == ComponentType.Table || xmlElement.Type == ComponentType.Tree || xmlElement.Type == ComponentType.ScrollGrid)
			{
				if (this.GetRecordByName(xmlElement.Name) == null)
				{
					string name = xmlElement.Name;
					XElement xelement = ComponentFactory.CreateRecord();
					xelement.SetAttributeValue("name", name.ToLower());
					this.Records.Add(xelement);
				}
			}
			else
			{
				this.AddRecordFieldTo(xmlElement, "Undefined");
			}
			if (xmlElement.Nodes != null)
			{
				foreach (XmlElement xmlElement2 in xmlElement.Nodes)
				{
					this.AddRecordField(xmlElement2);
				}
			}
		}

		// Token: 0x060000AC RID: 172 RVA: 0x00004800 File Offset: 0x00002A00
		private void AddRecordFieldTo(XmlElement componentModel, string recordName)
		{
			if (!ComponentFactory.IsIncludeProperties(componentModel.NodeName, "colName"))
			{
				return;
			}
			XElement xelement = this.GetRecordByName(recordName);
			if (xelement == null)
			{
				xelement = ComponentFactory.CreateRecord();
				xelement.SetAttributeValue("name", ("Undefined" != recordName) ? recordName.ToLower() : recordName);
				this.Records.Add(xelement);
			}
			XElement xelement2 = this.FindRecordFieldByAttributeValue("name", componentModel.Name);
			if (xelement2 == null)
			{
				xelement2 = ComponentFactory.CreateRecordField();
				xelement2.SetAttributeValue("name", componentModel.Name);
				componentModel.SetAttribute("fieldId", ScreenRecordManager.GetNewFieldIdRef(this.Records).ToString());
				xelement.Add(xelement2);
			}
			string attribute = componentModel.GetAttribute("fieldId");
			if (attribute != null)
			{
				xelement2.SetAttributeValue("fieldIdRef", componentModel.GetAttribute("fieldId"));
				xelement2.SetAttributeValue("fieldType", componentModel.GetAttribute("fieldType"));
				xelement2.SetAttributeValue("sqlTabName", componentModel.GetAttribute("sqlTabName"));
				xelement2.SetAttributeValue("colName", componentModel.GetAttribute("colName"));
			}
		}

		// Token: 0x060000AD RID: 173 RVA: 0x00004934 File Offset: 0x00002B34
		public void RemoveRecordField(XmlElement xmlElement)
		{
			this.RemoveRecordOrRecordField(xmlElement.Name);
			if (xmlElement.Nodes != null)
			{
				foreach (XmlElement xmlElement2 in xmlElement.Nodes)
				{
					this.RemoveRecordField(xmlElement2);
				}
			}
		}

		// Token: 0x060000AE RID: 174 RVA: 0x00004998 File Offset: 0x00002B98
		private void RemoveRecordOrRecordField(string fieldName)
		{
			foreach (XElement xelement in this.Records)
			{
				if (xelement.Attribute("name") != null && xelement.Attribute("name").Value.Equals(fieldName))
				{
					this.Records.Remove(xelement);
					break;
				}
				foreach (XElement xelement2 in xelement.Elements("RecordField"))
				{
					if (xelement2.Attribute("name") != null && xelement2.Attribute("name").Value.Equals(fieldName))
					{
						xelement2.Remove();
						return;
					}
				}
			}
		}

		// Token: 0x060000AF RID: 175 RVA: 0x00004AA4 File Offset: 0x00002CA4
		private XElement FindRecordFieldByAttributeValue(string attr, string value)
		{
			foreach (XElement xelement in this.Records)
			{
				foreach (XElement xelement2 in xelement.Elements())
				{
					string value2 = xelement2.Attribute(attr).Value;
					if (value.Equals(value2))
					{
						return xelement2;
					}
				}
			}
			return null;
		}

		// Token: 0x060000B0 RID: 176 RVA: 0x00004B50 File Offset: 0x00002D50
		private XElement GetRecordByName(string recordName)
		{
			foreach (XElement xelement in this.Records)
			{
				string value = xelement.Attribute("name").Value;
				if (recordName.Equals(value, StringComparison.CurrentCultureIgnoreCase))
				{
					return xelement;
				}
			}
			return null;
		}

		// Token: 0x060000B1 RID: 177 RVA: 0x00004BC4 File Offset: 0x00002DC4
		public void Clear()
		{
			if (this.Records != null)
			{
				this.Records.Clear();
			}
		}

		// Token: 0x060000B2 RID: 178 RVA: 0x00004BDC File Offset: 0x00002DDC
		internal static int GetNewFieldIdRef(List<XElement> Records)
		{
			List<int> list = new List<int>();
			foreach (XElement xelement in Records)
			{
				foreach (XElement xelement2 in xelement.Elements("RecordField"))
				{
					list.Add((int)short.Parse(xelement2.Attribute("fieldIdRef").Value));
				}
			}
			list.Sort();
			int num = 0;
			foreach (int num2 in list)
			{
				if (num == 0)
				{
					num = num2;
				}
				if (num2 - num > 1)
				{
					return num + 1;
				}
				num = num2;
			}
			return num + 1;
		}

		// Token: 0x0400003A RID: 58
		private List<XElement> Records;
	}
}
