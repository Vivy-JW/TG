using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Xml.Linq;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200012D RID: 301
	public class FglParameterNode : FglBaseNode
	{
		// Token: 0x170002CE RID: 718
		// (get) Token: 0x06000AB0 RID: 2736 RVA: 0x00034916 File Offset: 0x00032B16
		// (set) Token: 0x06000AB1 RID: 2737 RVA: 0x0003491E File Offset: 0x00032B1E
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

		// Token: 0x06000AB2 RID: 2738 RVA: 0x00034928 File Offset: 0x00032B28
		public FglParameterNode(FglComponent parent)
		{
			this.Parent = parent;
			this.Inputs = new ObservableCollection<FglParameter>();
			this.Returns = new ObservableCollection<FglParameter>();
			base.Status = CodeSpecStatus.NULL;
			this._attributes.Add("ver", "");
			this._attributes.Add("src", "");
		}

		// Token: 0x06000AB3 RID: 2739 RVA: 0x0003499B File Offset: 0x00032B9B
		public void Add(FglParameter param)
		{
			if (this.CheckExist(param))
			{
				throw new Exception("存在相同名稱資料");
			}
			if (param.IsReturn)
			{
				this.Returns.Add(param);
			}
			else
			{
				this.Inputs.Add(param);
			}
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000AB4 RID: 2740 RVA: 0x000349DC File Offset: 0x00032BDC
		public void Remove(FglParameter param)
		{
			if (!this.CheckExist(param))
			{
				throw new Exception("不存在此節點");
			}
			if (param.IsReturn)
			{
				this.Returns.Remove(param);
			}
			else
			{
				this.Inputs.Remove(param);
			}
			base.Status = CodeSpecStatus.MODIFY;
		}

		// Token: 0x06000AB5 RID: 2741 RVA: 0x00034A60 File Offset: 0x00032C60
		private bool CheckExist(FglParameter param)
		{
			if (param.IsReturn)
			{
				IEnumerable<FglParameter> enumerable = this.Returns.Where<FglParameter>((FglParameter p) => p.Name == param.Name);
				if (enumerable.Count<FglParameter>() > 0)
				{
					return true;
				}
			}
			else
			{
				IEnumerable<FglParameter> enumerable2 = this.Inputs.Where<FglParameter>((FglParameter p) => p.Name == param.Name);
				if (enumerable2.Count<FglParameter>() > 0)
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x06000AB6 RID: 2742 RVA: 0x00034AE1 File Offset: 0x00032CE1
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

		// Token: 0x06000AB7 RID: 2743 RVA: 0x00034B20 File Offset: 0x00032D20
		public static FglParameterNode Parse(FglComponent parent, XElement xelement)
		{
			FglParameterNode fglParameterNode = null;
			CodeSpecStatus codeSpecStatus = CodeSpecStatus.NULL;
			if (xelement.HasAttributes)
			{
				fglParameterNode = new FglParameterNode(parent);
				foreach (XAttribute xattribute in xelement.Attributes())
				{
					string text;
					if ((text = xattribute.Name.LocalName.ToLower()) != null)
					{
						if (!(text == "status"))
						{
							if (text == "cite_std")
							{
								if (xattribute.Value.Equals("Y", StringComparison.InvariantCultureIgnoreCase))
								{
									fglParameterNode.CiteStd = CiteEnum.Y;
									continue;
								}
								fglParameterNode.CiteStd = CiteEnum.N;
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
					fglParameterNode.SetAttribute(xattribute.Name.LocalName, xattribute.Value);
				}
				foreach (XElement xelement2 in xelement.Elements())
				{
					FglParameter fglParameter = FglParameter.Parse(fglParameterNode, xelement2);
					if (fglParameter != null)
					{
						fglParameterNode.Add(fglParameter);
					}
				}
			}
			fglParameterNode.Status = CodeSpecStatus.LOADED | codeSpecStatus;
			return fglParameterNode;
		}

		// Token: 0x06000AB8 RID: 2744 RVA: 0x00034C94 File Offset: 0x00032E94
		public XElement ToXElement()
		{
			XElement xelement = new XElement("param_rtn");
			xelement.SetAttributeValue("status", CodeSpecStatusToString.ToString(base.Status));
			xelement.SetAttributeValue("cite_std", this.CiteStd.ToString());
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			foreach (FglParameter fglParameter in this.Inputs)
			{
				xelement.Add(fglParameter.ToXElement());
			}
			foreach (FglParameter fglParameter2 in this.Returns)
			{
				xelement.Add(fglParameter2.ToXElement());
			}
			return xelement;
		}

		// Token: 0x040003FA RID: 1018
		private CiteEnum _citeStd = CiteEnum.N;

		// Token: 0x040003FB RID: 1019
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x040003FC RID: 1020
		public ObservableCollection<FglParameter> Inputs;

		// Token: 0x040003FD RID: 1021
		public ObservableCollection<FglParameter> Returns;
	}
}
