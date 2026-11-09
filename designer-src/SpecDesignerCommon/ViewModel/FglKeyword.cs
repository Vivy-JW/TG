using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200010B RID: 267
	public class FglKeyword : FglBaseObject, IDataErrorInfo
	{
		// Token: 0x17000278 RID: 632
		// (get) Token: 0x0600094B RID: 2379 RVA: 0x0002F274 File Offset: 0x0002D474
		// (set) Token: 0x0600094C RID: 2380 RVA: 0x0002F27C File Offset: 0x0002D47C
		private FglKeywordNode Parent { get; set; }

		// Token: 0x0600094D RID: 2381 RVA: 0x0002F285 File Offset: 0x0002D485
		public FglKeyword(FglKeywordNode parent)
		{
			this.Parent = parent;
			this.String = string.Empty;
		}

		// Token: 0x17000279 RID: 633
		// (get) Token: 0x0600094E RID: 2382 RVA: 0x0002F2B5 File Offset: 0x0002D4B5
		// (set) Token: 0x0600094F RID: 2383 RVA: 0x0002F2E8 File Offset: 0x0002D4E8
		public string Order
		{
			get
			{
				return this._order;
			}
			set
			{
				IEnumerable<FglKeyword> enumerable = this.Parent.FglKeywords.Where<FglKeyword>((FglKeyword p) => p.Order == value && p != this);
				if (enumerable.Count<FglKeyword>() > 0)
				{
					throw new Exception("順序值不可重複");
				}
				this._order = value;
				this.OnPropertySettingChanged("Order");
			}
		}

		// Token: 0x1700027A RID: 634
		// (get) Token: 0x06000950 RID: 2384 RVA: 0x0002F351 File Offset: 0x0002D551
		// (set) Token: 0x06000951 RID: 2385 RVA: 0x0002F359 File Offset: 0x0002D559
		public string String
		{
			get
			{
				return this._string;
			}
			set
			{
				this._string = value;
				this.OnPropertySettingChanged("String");
			}
		}

		// Token: 0x06000952 RID: 2386 RVA: 0x0002F370 File Offset: 0x0002D570
		private void OnPropertySettingChanged(string propertyName)
		{
			if ((this.Parent.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				this.Parent.Status = CodeSpecStatus.MODIFY;
				EventAggregatorManager.Get(this.Parent.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.Parent.ProgramKey);
			}
			base.OnPropertyChanged(propertyName);
		}

		// Token: 0x06000953 RID: 2387 RVA: 0x0002F3C8 File Offset: 0x0002D5C8
		public static FglKeyword Parse(FglKeywordNode parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			FglKeyword fglKeyword = new FglKeyword(parent);
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string localName;
				if ((localName = xattribute.Name.LocalName) != null)
				{
					if (localName == "order")
					{
						fglKeyword.Order = xattribute.Value;
						continue;
					}
					if (localName == "string")
					{
						fglKeyword.String = xattribute.Value;
						continue;
					}
				}
				fglKeyword._attributes.Add(xattribute.Name.LocalName, xattribute.Value);
			}
			return fglKeyword;
		}

		// Token: 0x06000954 RID: 2388 RVA: 0x0002F480 File Offset: 0x0002D680
		public XElement ToXElement()
		{
			XElement xelement = new XElement("key");
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			xelement.SetAttributeValue("order", this.Order);
			xelement.SetAttributeValue("string", this.String);
			return xelement;
		}

		// Token: 0x1700027B RID: 635
		// (get) Token: 0x06000955 RID: 2389 RVA: 0x0002F524 File Offset: 0x0002D724
		public string Error
		{
			get
			{
				return string.Join(Environment.NewLine, this._errorDic.Values);
			}
		}

		// Token: 0x1700027C RID: 636
		public string this[string columnName]
		{
			get
			{
				this._errorDic.Remove(columnName);
				if (columnName != null && columnName == "String" && string.IsNullOrWhiteSpace(this.String))
				{
					this._errorDic.Add(columnName, "關鍵字不可為空或空字串");
				}
				return this.Error;
			}
		}

		// Token: 0x04000371 RID: 881
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x04000372 RID: 882
		private string _order;

		// Token: 0x04000373 RID: 883
		private string _string;

		// Token: 0x04000374 RID: 884
		private Dictionary<string, string> _errorDic = new Dictionary<string, string>();
	}
}
