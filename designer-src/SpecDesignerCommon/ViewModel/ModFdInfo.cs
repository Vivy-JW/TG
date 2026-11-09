using System;
using System.Linq;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000029 RID: 41
	public class ModFdInfo
	{
		// Token: 0x06000158 RID: 344 RVA: 0x0000723E File Offset: 0x0000543E
		public ModFdInfo(string xmlString)
		{
			this._xmlString = xmlString;
		}

		// Token: 0x06000159 RID: 345 RVA: 0x00007284 File Offset: 0x00005484
		public bool IsIncludeAttribute(string type, string attrName)
		{
			if (this.modfd == null)
			{
				this.modfd = XElement.Parse(this._xmlString);
			}
			XElement xelement = (from a in this.modfd.Descendants("NodeInfo")
				where a.Attribute("mimeType").Value.Equals("modFD/" + type)
				select a).ElementAtOrDefault<XElement>(0);
			XAttribute xattribute = xelement.Attribute("properties");
			return xattribute != null && xattribute.Value.IndexOf(attrName) != -1;
		}

		// Token: 0x17000054 RID: 84
		// (get) Token: 0x0600015A RID: 346 RVA: 0x0000730C File Offset: 0x0000550C
		public string Content
		{
			get
			{
				return this._xmlString;
			}
		}

		// Token: 0x04000084 RID: 132
		private string _xmlString;

		// Token: 0x04000085 RID: 133
		private XElement modfd;
	}
}
