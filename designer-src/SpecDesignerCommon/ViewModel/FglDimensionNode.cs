using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000045 RID: 69
	public class FglDimensionNode : FglBaseNode
	{
		// Token: 0x17000088 RID: 136
		// (get) Token: 0x06000230 RID: 560 RVA: 0x00009A12 File Offset: 0x00007C12
		// (set) Token: 0x06000231 RID: 561 RVA: 0x00009A1A File Offset: 0x00007C1A
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

		// Token: 0x06000232 RID: 562 RVA: 0x00009A24 File Offset: 0x00007C24
		public FglDimensionNode(FglComponent parent)
		{
			this.Parent = parent;
			base.Status = CodeSpecStatus.NULL;
			this._attributes.Add("ver", "");
			this._attributes.Add("src", "");
		}

		// Token: 0x06000233 RID: 563 RVA: 0x00009A8C File Offset: 0x00007C8C
		public void Add(FglDimension dimension)
		{
			this.FglDimensions.Add(dimension);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000234 RID: 564 RVA: 0x00009AA1 File Offset: 0x00007CA1
		public void Remove(FglDimension dimension)
		{
			this.FglDimensions.Remove(dimension);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000235 RID: 565 RVA: 0x00009AB7 File Offset: 0x00007CB7
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

		// Token: 0x06000236 RID: 566 RVA: 0x00009AF4 File Offset: 0x00007CF4
		public static FglDimensionNode Parse(FglComponent parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			CodeSpecStatus codeSpecStatus = CodeSpecStatus.NULL;
			FglDimensionNode fglDimensionNode = new FglDimensionNode(parent);
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
								fglDimensionNode.CiteStd = CiteEnum.Y;
								continue;
							}
							fglDimensionNode.CiteStd = CiteEnum.N;
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
				fglDimensionNode.SetAttribute(xattribute.Name.LocalName, xattribute.Value);
			}
			foreach (XElement xelement2 in xelement.Elements())
			{
				string text;
				if ((text = xelement2.Name.LocalName.ToLower()) != null && text == "dimen")
				{
					FglDimension fglDimension = FglDimension.Parse(fglDimensionNode, xelement2);
					if (fglDimension != null)
					{
						fglDimensionNode.Add(fglDimension);
					}
				}
			}
			fglDimensionNode.Status = CodeSpecStatus.LOADED | codeSpecStatus;
			return fglDimensionNode;
		}

		// Token: 0x06000237 RID: 567 RVA: 0x00009C7C File Offset: 0x00007E7C
		public XElement ToXElement()
		{
			XElement xelement = new XElement("dimension");
			xelement.SetAttributeValue("status", CodeSpecStatusToString.ToString(base.Status));
			xelement.SetAttributeValue("cite_std", this.CiteStd.ToString());
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			foreach (FglDimension fglDimension in this.FglDimensions)
			{
				xelement.Add(fglDimension.ToXElement());
			}
			return xelement;
		}

		// Token: 0x040000D1 RID: 209
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x040000D2 RID: 210
		private CiteEnum _citeStd = CiteEnum.N;

		// Token: 0x040000D3 RID: 211
		public ObservableCollection<FglDimension> FglDimensions = new ObservableCollection<FglDimension>();
	}
}
