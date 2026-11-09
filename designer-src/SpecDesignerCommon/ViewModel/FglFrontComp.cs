using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200013F RID: 319
	public class FglFrontComp : FglBaseObject
	{
		// Token: 0x170002E9 RID: 745
		// (get) Token: 0x06000B1E RID: 2846 RVA: 0x00035D4C File Offset: 0x00033F4C
		// (set) Token: 0x06000B1F RID: 2847 RVA: 0x00035D54 File Offset: 0x00033F54
		public FglFrontCompNode Parent { get; set; }

		// Token: 0x06000B20 RID: 2848 RVA: 0x00035D5D File Offset: 0x00033F5D
		public FglFrontComp(FglFrontCompNode parent)
		{
			this.Parent = parent;
			this.Description = string.Empty;
		}

		// Token: 0x170002EA RID: 746
		// (get) Token: 0x06000B21 RID: 2849 RVA: 0x00035D82 File Offset: 0x00033F82
		// (set) Token: 0x06000B22 RID: 2850 RVA: 0x00035DB8 File Offset: 0x00033FB8
		public string Order
		{
			get
			{
				return this._order;
			}
			set
			{
				IEnumerable<FglFrontComp> enumerable = this.Parent.FglFrontComps.Where<FglFrontComp>((FglFrontComp p) => p.Order == value && p != this);
				if (enumerable.Count<FglFrontComp>() > 0)
				{
					throw new Exception("順序值不可重複");
				}
				if (string.IsNullOrWhiteSpace(value))
				{
					throw new Exception("順序值不可為空");
				}
				this._order = value;
				this.OnPropertySettingChanged("Order");
			}
		}

		// Token: 0x170002EB RID: 747
		// (get) Token: 0x06000B23 RID: 2851 RVA: 0x00035E39 File Offset: 0x00034039
		// (set) Token: 0x06000B24 RID: 2852 RVA: 0x00035E6C File Offset: 0x0003406C
		public string ID
		{
			get
			{
				return this._id;
			}
			set
			{
				IEnumerable<FglFrontComp> enumerable = this.Parent.FglFrontComps.Where<FglFrontComp>((FglFrontComp p) => p.ID == value && p != this);
				if (enumerable.Count<FglFrontComp>() > 0)
				{
					throw new Exception("ID不可重複");
				}
				this._id = value;
				this.OnPropertySettingChanged("ID");
				this.Description = CodeLibrariesHelper.This.GetDescription(this.ID);
			}
		}

		// Token: 0x170002EC RID: 748
		// (get) Token: 0x06000B25 RID: 2853 RVA: 0x00035EEB File Offset: 0x000340EB
		// (set) Token: 0x06000B26 RID: 2854 RVA: 0x00035EF3 File Offset: 0x000340F3
		public string Description
		{
			get
			{
				return this._description;
			}
			private set
			{
				this._description = value;
				this.OnPropertySettingChanged("Description");
			}
		}

		// Token: 0x06000B27 RID: 2855 RVA: 0x00035F08 File Offset: 0x00034108
		private void OnPropertySettingChanged(string propertyName)
		{
			if ((this.Parent.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				this.Parent.Status = CodeSpecStatus.MODIFY;
				EventAggregatorManager.Get(this.Parent.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.Parent.ProgramKey);
			}
			base.OnPropertyChanged(propertyName);
		}

		// Token: 0x06000B28 RID: 2856 RVA: 0x00035F60 File Offset: 0x00034160
		public static FglFrontComp Parse(FglFrontCompNode parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			FglFrontComp fglFrontComp = new FglFrontComp(parent);
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string localName;
				if ((localName = xattribute.Name.LocalName) != null)
				{
					if (localName == "order")
					{
						fglFrontComp.Order = xattribute.Value;
						continue;
					}
					if (localName == "id")
					{
						fglFrontComp.ID = xattribute.Value;
						continue;
					}
					if (localName == "desc")
					{
						fglFrontComp.Description = xattribute.Value;
						continue;
					}
				}
				fglFrontComp._attributes.Add(xattribute.Name.LocalName, xattribute.Value);
			}
			return fglFrontComp;
		}

		// Token: 0x06000B29 RID: 2857 RVA: 0x00036038 File Offset: 0x00034238
		public XElement ToXElement()
		{
			XElement xelement = new XElement("fcomp");
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			xelement.SetAttributeValue("order", this.Order);
			xelement.SetAttributeValue("id", this.ID);
			return xelement;
		}

		// Token: 0x04000448 RID: 1096
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x04000449 RID: 1097
		private string _order;

		// Token: 0x0400044A RID: 1098
		private string _id;

		// Token: 0x0400044B RID: 1099
		private string _description;
	}
}
