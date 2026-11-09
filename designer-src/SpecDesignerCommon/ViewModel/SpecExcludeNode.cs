using System;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000E5 RID: 229
	public class SpecExcludeNode
	{
		// Token: 0x17000223 RID: 547
		// (get) Token: 0x060007AE RID: 1966 RVA: 0x000225A0 File Offset: 0x000207A0
		// (set) Token: 0x060007AF RID: 1967 RVA: 0x000225A8 File Offset: 0x000207A8
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x060007B0 RID: 1968 RVA: 0x000225B1 File Offset: 0x000207B1
		public SpecExcludeNode()
		{
		}

		// Token: 0x060007B1 RID: 1969 RVA: 0x000225B9 File Offset: 0x000207B9
		public SpecExcludeNode(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return;
			}
			this.ProgramKey = key;
			this._source = source;
			this.GetStatusFromSource();
		}

		// Token: 0x17000224 RID: 548
		// (get) Token: 0x060007B2 RID: 1970 RVA: 0x000225D9 File Offset: 0x000207D9
		// (set) Token: 0x060007B3 RID: 1971 RVA: 0x000225E1 File Offset: 0x000207E1
		public SpecStatus Status
		{
			get
			{
				return this._status;
			}
			set
			{
				this._status = value;
			}
		}

		// Token: 0x060007B4 RID: 1972 RVA: 0x000225EC File Offset: 0x000207EC
		private void GetStatusFromSource()
		{
			string text;
			if (this.Source.Attribute("status") != null && (text = this.Source.Attribute("status").Value.ToLower()) != null)
			{
				if (text == "u")
				{
					this.Status = SpecStatus.MODIFY;
					return;
				}
				if (!(text == "d"))
				{
					return;
				}
				this.Status = SpecStatus.DELETE;
			}
		}

		// Token: 0x060007B5 RID: 1973 RVA: 0x00022660 File Offset: 0x00020860
		private void SetStatusToSource()
		{
			string text = "";
			if ((this.Status & SpecStatus.DELETE) == SpecStatus.DELETE)
			{
				text = "d";
			}
			else if ((this.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
			{
				text = "u";
			}
			else if ((this.Status & SpecStatus.CREATE) == SpecStatus.CREATE)
			{
				text = "c";
			}
			this.SetAttribute("status", text);
		}

		// Token: 0x17000225 RID: 549
		// (get) Token: 0x060007B6 RID: 1974 RVA: 0x000226B6 File Offset: 0x000208B6
		public XElement Source
		{
			get
			{
				return this._source;
			}
		}

		// Token: 0x060007B7 RID: 1975 RVA: 0x000226BE File Offset: 0x000208BE
		public void SetAttribute(string key, string value)
		{
			if (value == null)
			{
				value = "";
			}
			this.Source.SetAttributeValue(key, value);
		}

		// Token: 0x060007B8 RID: 1976 RVA: 0x000226DC File Offset: 0x000208DC
		public string GetAttribute(string key)
		{
			if (this.Source != null && this.Source.Attribute(key) != null)
			{
				return this.Source.Attribute(key).Value;
			}
			return null;
		}

		// Token: 0x060007B9 RID: 1977 RVA: 0x00022711 File Offset: 0x00020911
		public XElement ToXml()
		{
			if (SpecStatus.CREATE == this.Status)
			{
				return null;
			}
			this.SetStatusToSource();
			return this.Source;
		}

		// Token: 0x060007BA RID: 1978 RVA: 0x0002272C File Offset: 0x0002092C
		public override string ToString()
		{
			XElement xelement = this.ToXml();
			if (xelement != null)
			{
				return xelement.ToString();
			}
			return null;
		}

		// Token: 0x060007BB RID: 1979 RVA: 0x0002274C File Offset: 0x0002094C
		internal static SpecExcludeNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("widget", new object[]
			{
				new XAttribute("name", name),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecExcludeNode(info.Key, xelement);
		}

		// Token: 0x060007BC RID: 1980 RVA: 0x000227AD File Offset: 0x000209AD
		internal void SetName(FormSpecModel formSpecModel, string value)
		{
			this.SetAttribute("name", value);
		}

		// Token: 0x040002B7 RID: 695
		private SpecStatus _status;

		// Token: 0x040002B8 RID: 696
		private XElement _source;
	}
}
