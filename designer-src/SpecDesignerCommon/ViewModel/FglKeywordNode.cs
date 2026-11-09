using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200010A RID: 266
	public class FglKeywordNode : FglBaseNode
	{
		// Token: 0x17000277 RID: 631
		// (get) Token: 0x06000943 RID: 2371 RVA: 0x0002EF12 File Offset: 0x0002D112
		// (set) Token: 0x06000944 RID: 2372 RVA: 0x0002EF1A File Offset: 0x0002D11A
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

		// Token: 0x06000945 RID: 2373 RVA: 0x0002EF24 File Offset: 0x0002D124
		public FglKeywordNode(FglComponent parent)
		{
			this.Parent = parent;
			base.Status = CodeSpecStatus.NULL;
			this._attributes.Add("ver", "");
			this._attributes.Add("src", "");
		}

		// Token: 0x06000946 RID: 2374 RVA: 0x0002EF8C File Offset: 0x0002D18C
		public void Add(FglKeyword keyword)
		{
			this.FglKeywords.Add(keyword);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000947 RID: 2375 RVA: 0x0002EFA1 File Offset: 0x0002D1A1
		public void Remove(FglKeyword keyword)
		{
			this.FglKeywords.Remove(keyword);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000948 RID: 2376 RVA: 0x0002EFB7 File Offset: 0x0002D1B7
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

		// Token: 0x06000949 RID: 2377 RVA: 0x0002EFF4 File Offset: 0x0002D1F4
		public static FglKeywordNode Parse(FglComponent parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			CodeSpecStatus codeSpecStatus = CodeSpecStatus.NULL;
			FglKeywordNode fglKeywordNode = new FglKeywordNode(parent);
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
								fglKeywordNode.CiteStd = CiteEnum.Y;
								continue;
							}
							fglKeywordNode.CiteStd = CiteEnum.N;
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
				fglKeywordNode.SetAttribute(xattribute.Name.LocalName, xattribute.Value);
			}
			foreach (XElement xelement2 in xelement.Elements())
			{
				string text;
				if ((text = xelement2.Name.LocalName.ToLower()) != null && text == "key")
				{
					FglKeyword fglKeyword = FglKeyword.Parse(fglKeywordNode, xelement2);
					if (fglKeyword != null)
					{
						fglKeywordNode.Add(fglKeyword);
					}
				}
			}
			fglKeywordNode.Status = CodeSpecStatus.LOADED | codeSpecStatus;
			return fglKeywordNode;
		}

		// Token: 0x0600094A RID: 2378 RVA: 0x0002F17C File Offset: 0x0002D37C
		public XElement ToXElement()
		{
			XElement xelement = new XElement("key_word");
			xelement.SetAttributeValue("status", CodeSpecStatusToString.ToString(base.Status));
			xelement.SetAttributeValue("cite_std", this.CiteStd.ToString());
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			foreach (FglKeyword fglKeyword in this.FglKeywords)
			{
				xelement.Add(fglKeyword.ToXElement());
			}
			return xelement;
		}

		// Token: 0x0400036E RID: 878
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x0400036F RID: 879
		private CiteEnum _citeStd = CiteEnum.N;

		// Token: 0x04000370 RID: 880
		public ObservableCollection<FglKeyword> FglKeywords = new ObservableCollection<FglKeyword>();
	}
}
