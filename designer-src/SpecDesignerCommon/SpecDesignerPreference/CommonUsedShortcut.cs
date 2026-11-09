using System;
using System.Xml.Linq;

namespace SpecDesignerPreference
{
	// Token: 0x02000088 RID: 136
	public class CommonUsedShortcut
	{
		// Token: 0x170001A0 RID: 416
		// (get) Token: 0x060005A2 RID: 1442 RVA: 0x0001A1DD File Offset: 0x000183DD
		// (set) Token: 0x060005A3 RID: 1443 RVA: 0x0001A1E5 File Offset: 0x000183E5
		public string Name { get; set; }

		// Token: 0x170001A1 RID: 417
		// (get) Token: 0x060005A4 RID: 1444 RVA: 0x0001A1EE File Offset: 0x000183EE
		// (set) Token: 0x060005A5 RID: 1445 RVA: 0x0001A1F6 File Offset: 0x000183F6
		public string Sort { get; set; }

		// Token: 0x170001A2 RID: 418
		// (get) Token: 0x060005A6 RID: 1446 RVA: 0x0001A1FF File Offset: 0x000183FF
		// (set) Token: 0x060005A7 RID: 1447 RVA: 0x0001A207 File Offset: 0x00018407
		public string Enabled { get; set; }

		// Token: 0x170001A3 RID: 419
		// (get) Token: 0x060005A8 RID: 1448 RVA: 0x0001A210 File Offset: 0x00018410
		// (set) Token: 0x060005A9 RID: 1449 RVA: 0x0001A218 File Offset: 0x00018418
		public string Desc { get; set; }

		// Token: 0x060005AA RID: 1450 RVA: 0x0001A221 File Offset: 0x00018421
		public CommonUsedShortcut(string name, string sort, string enabled, string desc)
		{
			this.Name = name;
			this.Sort = sort;
			this.Enabled = enabled;
			this.Desc = desc;
		}

		// Token: 0x170001A4 RID: 420
		// (get) Token: 0x060005AB RID: 1451 RVA: 0x0001A246 File Offset: 0x00018446
		public string Title
		{
			get
			{
				if (!string.IsNullOrEmpty(this.Desc))
				{
					return string.Format("{0}({1})", this.Name, this.Desc);
				}
				return this.Name;
			}
		}

		// Token: 0x060005AC RID: 1452 RVA: 0x0001A274 File Offset: 0x00018474
		public static CommonUsedShortcut Parse(XElement source)
		{
			if (source.Attribute("Name") == null || source.Attribute("Sort") == null)
			{
				return null;
			}
			string value = source.Attribute("Name").Value;
			string value2 = source.Attribute("Sort").Value;
			string text = ((source.Attribute("Enabled") == null) ? "N" : source.Attribute("Enabled").Value);
			string text2 = ((source.Attribute("Desc") == null) ? string.Empty : source.Attribute("Desc").Value);
			return new CommonUsedShortcut(value, value2, text, text2);
		}

		// Token: 0x060005AD RID: 1453 RVA: 0x0001A33C File Offset: 0x0001853C
		public XElement ToXML()
		{
			XElement xelement = new XElement("Item");
			xelement.SetAttributeValue("Name", this.Name);
			xelement.SetAttributeValue("Sort", this.Sort);
			xelement.SetAttributeValue("Enabled", this.Enabled);
			xelement.SetAttributeValue("Desc", this.Desc);
			return xelement;
		}
	}
}
