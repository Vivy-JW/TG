using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000047 RID: 71
	public class FglDimension : FglBaseObject
	{
		// Token: 0x17000089 RID: 137
		// (get) Token: 0x0600023F RID: 575 RVA: 0x00009E21 File Offset: 0x00008021
		// (set) Token: 0x06000240 RID: 576 RVA: 0x00009E29 File Offset: 0x00008029
		private FglDimensionNode Parent { get; set; }

		// Token: 0x06000241 RID: 577 RVA: 0x00009E32 File Offset: 0x00008032
		public FglDimension(FglDimensionNode parent)
		{
			this.Parent = parent;
		}

		// Token: 0x1700008A RID: 138
		// (get) Token: 0x06000242 RID: 578 RVA: 0x00009E4C File Offset: 0x0000804C
		// (set) Token: 0x06000243 RID: 579 RVA: 0x00009E94 File Offset: 0x00008094
		public string No
		{
			get
			{
				return this._no;
			}
			set
			{
				IEnumerable<FglDimension> enumerable = this.Parent.FglDimensions.Where<FglDimension>((FglDimension p) => p.No == value && p != this);
				if (enumerable.Count<FglDimension>() > 0)
				{
					throw new Exception("編號不可重複");
				}
				if (string.IsNullOrWhiteSpace(value))
				{
					throw new Exception("編號不可為空");
				}
				DimensionOption dimensionOption = SettingManager.Get().Dimensions.Where<DimensionOption>((DimensionOption di) => di.No == value).ElementAtOrDefault<DimensionOption>(0);
				if (dimensionOption == null)
				{
					throw new Exception(string.Format("無相對維度項目：{0}", value));
				}
				this._no = value;
				this.OnPropertySettingChanged("No");
				this.Description = dimensionOption.Description;
				this.Class = dimensionOption.Class[0].Name;
			}
		}

		// Token: 0x1700008B RID: 139
		// (get) Token: 0x06000244 RID: 580 RVA: 0x00009F73 File Offset: 0x00008173
		// (set) Token: 0x06000245 RID: 581 RVA: 0x00009F7B File Offset: 0x0000817B
		public string Description
		{
			get
			{
				return this._description;
			}
			set
			{
				this._description = value;
				this.OnPropertySettingChanged("Description");
			}
		}

		// Token: 0x1700008C RID: 140
		// (get) Token: 0x06000246 RID: 582 RVA: 0x00009F8F File Offset: 0x0000818F
		// (set) Token: 0x06000247 RID: 583 RVA: 0x00009FC0 File Offset: 0x000081C0
		public string Class
		{
			get
			{
				return this._class;
			}
			set
			{
				this._class = value;
				this.OnPropertySettingChanged("Class");
				DimensionOption dimensionOption = SettingManager.Get().Dimensions.Where<DimensionOption>((DimensionOption di) => di.No == this._no).ElementAtOrDefault<DimensionOption>(0);
				DimensionClassOption dimensionClassOption = dimensionOption.Class.Where<DimensionClassOption>((DimensionClassOption c) => c.Name == this._class).ElementAtOrDefault<DimensionClassOption>(0);
				this.ClassValue = ((dimensionClassOption == null) ? string.Empty : dimensionClassOption.Description);
			}
		}

		// Token: 0x1700008D RID: 141
		// (get) Token: 0x06000248 RID: 584 RVA: 0x0000A035 File Offset: 0x00008235
		// (set) Token: 0x06000249 RID: 585 RVA: 0x0000A03D File Offset: 0x0000823D
		public string ClassValue
		{
			get
			{
				return this._classValue;
			}
			set
			{
				this._classValue = value;
				this.OnPropertySettingChanged("ClassValue");
			}
		}

		// Token: 0x0600024A RID: 586 RVA: 0x0000A054 File Offset: 0x00008254
		private void OnPropertySettingChanged(string propertyName)
		{
			if ((this.Parent.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				this.Parent.Status = CodeSpecStatus.MODIFY;
				EventAggregatorManager.Get(this.Parent.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.Parent.ProgramKey);
			}
			base.OnPropertyChanged(propertyName);
		}

		// Token: 0x0600024B RID: 587 RVA: 0x0000A0AC File Offset: 0x000082AC
		public static FglDimension Parse(FglDimensionNode parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			FglDimension fglDimension = new FglDimension(parent);
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string localName;
				if ((localName = xattribute.Name.LocalName) != null)
				{
					if (localName == "no")
					{
						fglDimension.No = xattribute.Value;
						continue;
					}
					if (localName == "desc")
					{
						fglDimension.Description = xattribute.Value;
						continue;
					}
					if (localName == "class")
					{
						fglDimension.Class = xattribute.Value;
						continue;
					}
					if (localName == "class_value")
					{
						fglDimension.ClassValue = xattribute.Value;
						continue;
					}
				}
				fglDimension._attributes.Add(xattribute.Name.LocalName, xattribute.Value);
			}
			return fglDimension;
		}

		// Token: 0x0600024C RID: 588 RVA: 0x0000A1A0 File Offset: 0x000083A0
		public XElement ToXElement()
		{
			XElement xelement = new XElement("dimen");
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			xelement.SetAttributeValue("no", this.No);
			xelement.SetAttributeValue("desc", this.Description);
			xelement.SetAttributeValue("class", this.Class);
			xelement.SetAttributeValue("class_value", this.ClassValue);
			return xelement;
		}

		// Token: 0x040000D5 RID: 213
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x040000D6 RID: 214
		private string _no;

		// Token: 0x040000D7 RID: 215
		private string _description;

		// Token: 0x040000D8 RID: 216
		private string _class;

		// Token: 0x040000D9 RID: 217
		private string _classValue;
	}
}
