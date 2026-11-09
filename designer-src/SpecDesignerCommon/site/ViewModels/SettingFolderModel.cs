using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.Site.Behaviors;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x02000035 RID: 53
	public class SettingFolderModel : SettingModelBase, IDragable, IDropable
	{
		// Token: 0x1700006C RID: 108
		// (get) Token: 0x060001BC RID: 444 RVA: 0x00008300 File Offset: 0x00006500
		// (set) Token: 0x060001BD RID: 445 RVA: 0x00008308 File Offset: 0x00006508
		public ObservableCollection<SettingModelBase> Settings { get; internal set; }

		// Token: 0x060001BE RID: 446 RVA: 0x00008311 File Offset: 0x00006511
		public SettingFolderModel()
		{
		}

		// Token: 0x060001BF RID: 447 RVA: 0x00008319 File Offset: 0x00006519
		public SettingFolderModel(XElement element)
		{
			this.ParseChildren(element);
			this.ParseAttributes(element);
		}

		// Token: 0x060001C0 RID: 448 RVA: 0x00008330 File Offset: 0x00006530
		private void ParseAttributes(XElement element)
		{
			base.Name = ((element.Attribute("name") == null) ? "" : element.Attribute("name").Value);
			base.Order = SettingModelBase.ParseOrder(element);
		}

		// Token: 0x060001C1 RID: 449 RVA: 0x0000837D File Offset: 0x0000657D
		internal void Remove(SettingModelBase settingModelBase)
		{
			this.Settings.Remove(settingModelBase);
			this.ReOrder();
		}

		// Token: 0x060001C2 RID: 450 RVA: 0x00008394 File Offset: 0x00006594
		internal void ParseChildren(XElement element)
		{
			IEnumerable<XElement> enumerable = element.Elements("Folder");
			List<SettingModelBase> list = new List<SettingModelBase>();
			foreach (XElement xelement in enumerable)
			{
				list.Add(new SettingFolderModel(xelement)
				{
					Parent = this
				});
			}
			enumerable = element.Elements("Setting");
			foreach (XElement xelement2 in enumerable)
			{
				list.Add(new SettingModel(xelement2)
				{
					Parent = this
				});
			}
			this.Settings = new ObservableCollection<SettingModelBase>(list);
			this.ReOrder();
			base.NotifyPropertyChanged("Settings");
		}

		// Token: 0x060001C3 RID: 451 RVA: 0x00008484 File Offset: 0x00006684
		public void Insert(SettingModelBase model)
		{
			this.Insert(model, this.Settings.Count);
		}

		// Token: 0x060001C4 RID: 452 RVA: 0x00008498 File Offset: 0x00006698
		public void Insert(SettingModelBase model, int order)
		{
			model.Remove();
			model.Parent = this;
			if (order < 0)
			{
				order = 0;
			}
			this.Settings.Insert(order, model);
			this.ReOrder();
		}

		// Token: 0x060001C5 RID: 453 RVA: 0x000084C4 File Offset: 0x000066C4
		private void ReOrder()
		{
			for (int i = 0; i < this.Settings.Count; i++)
			{
				SettingModelBase settingModelBase = this.Settings[i];
				settingModelBase.Order = i;
			}
		}

		// Token: 0x060001C6 RID: 454 RVA: 0x000084FC File Offset: 0x000066FC
		internal static SettingFolderModel Create()
		{
			return new SettingFolderModel(XElement.Parse(SettingFolderModel.xml, LoadOptions.None));
		}

		// Token: 0x060001C7 RID: 455 RVA: 0x0000851C File Offset: 0x0000671C
		public override SettingModelBase Clone()
		{
			return new SettingFolderModel(XElement.Parse(SettingFolderModel.xml, LoadOptions.None))
			{
				Name = base.Name
			};
		}

		// Token: 0x060001C8 RID: 456 RVA: 0x00008548 File Offset: 0x00006748
		public override XElement ToXML()
		{
			XElement xelement = new XElement("Folder", new object[]
			{
				new XAttribute("name", base.Name),
				new XAttribute("order", base.Order)
			});
			foreach (SettingModelBase settingModelBase in this.Settings)
			{
				xelement.Add(settingModelBase.ToXML());
			}
			return xelement;
		}

		// Token: 0x1700006D RID: 109
		// (get) Token: 0x060001C9 RID: 457 RVA: 0x000085E8 File Offset: 0x000067E8
		public Type DragType
		{
			get
			{
				return typeof(SettingModelBase);
			}
		}

		// Token: 0x1700006E RID: 110
		// (get) Token: 0x060001CA RID: 458 RVA: 0x000085F4 File Offset: 0x000067F4
		public Type AllowType
		{
			get
			{
				return typeof(SettingModelBase);
			}
		}

		// Token: 0x060001CB RID: 459 RVA: 0x00008600 File Offset: 0x00006800
		public void DropOver(DragEventArgs dragEvetnArgs)
		{
		}

		// Token: 0x060001CC RID: 460 RVA: 0x00008604 File Offset: 0x00006804
		public void Drop(IDragable drag)
		{
			SettingModelBase settingModelBase = drag as SettingModelBase;
			this.Insert(settingModelBase);
		}

		// Token: 0x060001CD RID: 461 RVA: 0x00008620 File Offset: 0x00006820
		public bool CanDrop(IDragable drag)
		{
			if (drag == this)
			{
				return false;
			}
			SettingFolderModel settingFolderModel = this;
			while ((settingFolderModel = settingFolderModel.Parent) != null)
			{
				if (settingFolderModel == drag)
				{
					return false;
				}
			}
			return true;
		}

		// Token: 0x060001CE RID: 462 RVA: 0x00008648 File Offset: 0x00006848
		public override SettingModel GetSettingByUid(string uid)
		{
			foreach (SettingModelBase settingModelBase in this.Settings)
			{
				SettingModel settingByUid = settingModelBase.GetSettingByUid(uid);
				if (settingByUid != null)
				{
					return settingByUid;
				}
			}
			return null;
		}

		// Token: 0x040000A8 RID: 168
		private static string xml = "<Folder name='New Folder' />";
	}
}
