using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200013E RID: 318
	public class FglFrontCompNode : FglBaseNode
	{
		// Token: 0x170002E8 RID: 744
		// (get) Token: 0x06000B16 RID: 2838 RVA: 0x000359EB File Offset: 0x00033BEB
		// (set) Token: 0x06000B17 RID: 2839 RVA: 0x000359F3 File Offset: 0x00033BF3
		public CiteEnum CiteStd
		{
			get
			{
				return this._citeStd;
			}
			set
			{
				this._citeStd = value;
			}
		}

		// Token: 0x06000B18 RID: 2840 RVA: 0x000359FC File Offset: 0x00033BFC
		public FglFrontCompNode(FglComponent parent)
		{
			this.Parent = parent;
			base.Status = CodeSpecStatus.NULL;
			this._attributes.Add("ver", "");
			this._attributes.Add("src", "");
		}

		// Token: 0x06000B19 RID: 2841 RVA: 0x00035A64 File Offset: 0x00033C64
		public void Add(FglFrontComp keyword)
		{
			this.FglFrontComps.Add(keyword);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000B1A RID: 2842 RVA: 0x00035A79 File Offset: 0x00033C79
		public void Remove(FglFrontComp keyword)
		{
			this.FglFrontComps.Remove(keyword);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000B1B RID: 2843 RVA: 0x00035A8F File Offset: 0x00033C8F
		public void SetAttribute(string key, string value)
		{
			if (this._attributes == null)
			{
				this._attributes = new Dictionary<string, string>();
			}
			if (this._attributes.ContainsKey(key))
			{
				this._attributes.Remove(key);
			}
			this._attributes.Add(key, value);
		}

		// Token: 0x06000B1C RID: 2844 RVA: 0x00035ACC File Offset: 0x00033CCC
		public static FglFrontCompNode Parse(FglComponent parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			CodeSpecStatus codeSpecStatus = CodeSpecStatus.NULL;
			FglFrontCompNode fglFrontCompNode = new FglFrontCompNode(parent);
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string localName;
				if ((localName = xattribute.Name.LocalName) != null)
				{
					if (!(localName == "status"))
					{
						if (localName == "cite_std")
						{
							if (xattribute.Value.Equals("Y", StringComparison.InvariantCultureIgnoreCase))
							{
								fglFrontCompNode.CiteStd = CiteEnum.Y;
								continue;
							}
							fglFrontCompNode.CiteStd = CiteEnum.N;
							continue;
						}
					}
					else
					{
						if (xattribute.Value.Equals("d", StringComparison.InvariantCultureIgnoreCase))
						{
							codeSpecStatus = CodeSpecStatus.DELETE;
							continue;
						}
						if (xattribute.Value.Equals("u", StringComparison.InvariantCultureIgnoreCase))
						{
							codeSpecStatus = CodeSpecStatus.MODIFY;
							continue;
						}
						if (string.IsNullOrWhiteSpace(xattribute.Value))
						{
							codeSpecStatus = CodeSpecStatus.NULL;
							continue;
						}
						continue;
					}
				}
				fglFrontCompNode.SetAttribute(xattribute.Name.LocalName, xattribute.Value);
			}
			foreach (XElement xelement2 in xelement.Elements())
			{
				string text;
				if ((text = xelement2.Name.LocalName.ToLower()) != null && text == "fcomp")
				{
					FglFrontComp fglFrontComp = FglFrontComp.Parse(fglFrontCompNode, xelement2);
					if (fglFrontComp != null)
					{
						fglFrontCompNode.Add(fglFrontComp);
					}
				}
			}
			fglFrontCompNode.Status = CodeSpecStatus.LOADED | codeSpecStatus;
			return fglFrontCompNode;
		}

		// Token: 0x06000B1D RID: 2845 RVA: 0x00035C54 File Offset: 0x00033E54
		public XElement ToXElement()
		{
			XElement xelement = new XElement("front_comp");
			xelement.SetAttributeValue("status", CodeSpecStatusToString.ToString(base.Status));
			xelement.SetAttributeValue("cite_std", this.CiteStd.ToString());
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			foreach (FglFrontComp fglFrontComp in this.FglFrontComps)
			{
				xelement.Add(fglFrontComp.ToXElement());
			}
			return xelement;
		}

		// Token: 0x04000445 RID: 1093
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x04000446 RID: 1094
		private CiteEnum _citeStd = CiteEnum.N;

		// Token: 0x04000447 RID: 1095
		public ObservableCollection<FglFrontComp> FglFrontComps = new ObservableCollection<FglFrontComp>();
	}
}
