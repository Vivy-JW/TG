using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200006F RID: 111
	public class FglTestNode : FglBaseNode
	{
		// Token: 0x17000122 RID: 290
		// (get) Token: 0x06000444 RID: 1092 RVA: 0x0001375E File Offset: 0x0001195E
		// (set) Token: 0x06000445 RID: 1093 RVA: 0x00013766 File Offset: 0x00011966
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

		// Token: 0x06000446 RID: 1094 RVA: 0x00013770 File Offset: 0x00011970
		public FglTestNode(FglComponent parent)
		{
			this.Parent = parent;
			base.Status = CodeSpecStatus.NULL;
			this._attributes.Add("ver", "");
			this._attributes.Add("src", "");
		}

		// Token: 0x06000447 RID: 1095 RVA: 0x000137D8 File Offset: 0x000119D8
		public void Add(FglTest test)
		{
			this.FglTests.Add(test);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000448 RID: 1096 RVA: 0x000137ED File Offset: 0x000119ED
		public void Remove(FglTest keyword)
		{
			this.FglTests.Remove(keyword);
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000449 RID: 1097 RVA: 0x00013803 File Offset: 0x00011A03
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

		// Token: 0x0600044A RID: 1098 RVA: 0x00013840 File Offset: 0x00011A40
		public static FglTestNode Parse(FglComponent parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			CodeSpecStatus codeSpecStatus = CodeSpecStatus.NULL;
			FglTestNode fglTestNode = new FglTestNode(parent);
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
								fglTestNode.CiteStd = CiteEnum.Y;
								continue;
							}
							fglTestNode.CiteStd = CiteEnum.N;
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
							fglTestNode.Status = CodeSpecStatus.MODIFY;
							continue;
						}
						if (string.IsNullOrWhiteSpace(xattribute.Value))
						{
							fglTestNode.Status = CodeSpecStatus.NULL;
							continue;
						}
						continue;
					}
				}
				fglTestNode.SetAttribute(xattribute.Name.LocalName, xattribute.Value);
			}
			foreach (XElement xelement2 in xelement.Elements())
			{
				string text;
				if ((text = xelement2.Name.LocalName.ToLower()) != null && text == "scenario")
				{
					FglTest fglTest = FglTest.Parse(fglTestNode, xelement2);
					if (fglTest != null)
					{
						fglTestNode.Add(fglTest);
					}
				}
			}
			fglTestNode.Status = CodeSpecStatus.LOADED | codeSpecStatus;
			return fglTestNode;
		}

		// Token: 0x0600044B RID: 1099 RVA: 0x000139D4 File Offset: 0x00011BD4
		public XElement ToXElement()
		{
			XElement xelement = new XElement("test");
			xelement.SetAttributeValue("status", CodeSpecStatusToString.ToString(base.Status));
			xelement.SetAttributeValue("cite_std", this.CiteStd.ToString());
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			foreach (FglTest fglTest in this.FglTests)
			{
				xelement.Add(fglTest.ToXElement());
			}
			return xelement;
		}

		// Token: 0x040001A9 RID: 425
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x040001AA RID: 426
		private CiteEnum _citeStd = CiteEnum.N;

		// Token: 0x040001AB RID: 427
		public ObservableCollection<FglTest> FglTests = new ObservableCollection<FglTest>();
	}
}
