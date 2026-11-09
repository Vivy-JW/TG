using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200011F RID: 287
	public class FglSpecContent : FglBaseNode, INotifyPropertyChanged
	{
		// Token: 0x170002A7 RID: 679
		// (get) Token: 0x06000A14 RID: 2580 RVA: 0x0003212A File Offset: 0x0003032A
		// (set) Token: 0x06000A15 RID: 2581 RVA: 0x00032132 File Offset: 0x00030332
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

		// Token: 0x170002A8 RID: 680
		// (get) Token: 0x06000A16 RID: 2582 RVA: 0x0003213B File Offset: 0x0003033B
		// (set) Token: 0x06000A17 RID: 2583 RVA: 0x00032143 File Offset: 0x00030343
		public string Content
		{
			get
			{
				return this._content;
			}
			set
			{
				if (this._content == value)
				{
					return;
				}
				this._content = value;
				this.OnPropertyChanged("Content");
			}
		}

		// Token: 0x06000A18 RID: 2584 RVA: 0x00032168 File Offset: 0x00030368
		public FglSpecContent(FglComponent parent)
		{
			this.Parent = parent;
			this.Content = string.Empty;
			base.Status = CodeSpecStatus.NULL;
			this._attributes.Add("ver", "");
			this._attributes.Add("src", "");
		}

		// Token: 0x06000A19 RID: 2585 RVA: 0x000321D0 File Offset: 0x000303D0
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

		// Token: 0x06000A1A RID: 2586 RVA: 0x00032210 File Offset: 0x00030410
		public static FglSpecContent Parse(FglComponent parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			CodeSpecStatus codeSpecStatus = CodeSpecStatus.NULL;
			FglSpecContent fglSpecContent = new FglSpecContent(parent);
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
								fglSpecContent.CiteStd = CiteEnum.Y;
								continue;
							}
							fglSpecContent.CiteStd = CiteEnum.N;
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
				fglSpecContent.SetAttribute(xattribute.Name.LocalName, xattribute.Value);
			}
			fglSpecContent.Content = xelement.Value;
			fglSpecContent.Status = CodeSpecStatus.LOADED | codeSpecStatus;
			return fglSpecContent;
		}

		// Token: 0x06000A1B RID: 2587 RVA: 0x0003232C File Offset: 0x0003052C
		public XElement ToXElement()
		{
			XElement xelement = new XElement("spec");
			xelement.SetAttributeValue("status", CodeSpecStatusToString.ToString(base.Status));
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			xelement.Add(new XCData(this.Content));
			return xelement;
		}

		// Token: 0x14000023 RID: 35
		// (add) Token: 0x06000A1C RID: 2588 RVA: 0x000323D0 File Offset: 0x000305D0
		// (remove) Token: 0x06000A1D RID: 2589 RVA: 0x00032408 File Offset: 0x00030608
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000A1E RID: 2590 RVA: 0x00032440 File Offset: 0x00030640
		public void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
			if ((base.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				base.Status = CodeSpecStatus.MODIFY | base.Status;
				EventAggregatorManager.Get(this.Parent.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.Parent.ProgramKey);
			}
		}

		// Token: 0x040003CB RID: 971
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x040003CC RID: 972
		private CiteEnum _citeStd = CiteEnum.N;

		// Token: 0x040003CD RID: 973
		private string _content;
	}
}
