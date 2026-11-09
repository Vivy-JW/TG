using System;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.IO;
using System.Linq;
using System.Windows;
using System.Xml.Linq;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x020000E8 RID: 232
	public class SiteManagerModel : INotifyPropertyChanged
	{
		// Token: 0x17000226 RID: 550
		// (get) Token: 0x060007C5 RID: 1989 RVA: 0x00022E19 File Offset: 0x00021019
		// (set) Token: 0x060007C6 RID: 1990 RVA: 0x00022E21 File Offset: 0x00021021
		private SettingFolderModel Root { get; set; }

		// Token: 0x17000227 RID: 551
		// (get) Token: 0x060007C7 RID: 1991 RVA: 0x00022E2A File Offset: 0x0002102A
		// (set) Token: 0x060007C8 RID: 1992 RVA: 0x00022E32 File Offset: 0x00021032
		public ObservableCollection<SettingModelBase> Settings { get; private set; }

		// Token: 0x060007C9 RID: 1993 RVA: 0x00022E3B File Offset: 0x0002103B
		public SiteManagerModel(string xml)
		{
			this.Parse(xml);
		}

		// Token: 0x060007CA RID: 1994 RVA: 0x00022E4C File Offset: 0x0002104C
		public void Parse(string xml)
		{
			XElement xelement = XElement.Parse(xml, LoadOptions.None);
			this.Root = new SettingFolderModel(xelement)
			{
				Name = (Application.Current.FindResource("Site_ConnectionList") as string)
			};
			this.Settings = new ObservableCollection<SettingModelBase>();
			this.Settings.Add(this.Root);
			this.NotifyPropertyChanged("Settings");
		}

		// Token: 0x1400001C RID: 28
		// (add) Token: 0x060007CB RID: 1995 RVA: 0x00022EB0 File Offset: 0x000210B0
		// (remove) Token: 0x060007CC RID: 1996 RVA: 0x00022EE8 File Offset: 0x000210E8
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060007CD RID: 1997 RVA: 0x00022F1D File Offset: 0x0002111D
		private void NotifyPropertyChanged(string info)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(info));
			}
		}

		// Token: 0x17000228 RID: 552
		// (get) Token: 0x060007CE RID: 1998 RVA: 0x00022F3C File Offset: 0x0002113C
		public SettingModelBase FirstSetting
		{
			get
			{
				SettingModelBase settingModelBase = this.Root.Settings.FirstOrDefault<SettingModelBase>();
				if (settingModelBase is SettingFolderModel)
				{
					return (settingModelBase as SettingFolderModel).Settings.FirstOrDefault<SettingModelBase>();
				}
				return settingModelBase;
			}
		}

		// Token: 0x060007CF RID: 1999 RVA: 0x00022F74 File Offset: 0x00021174
		public XElement ToXML()
		{
			XElement xelement = new XElement("Settings");
			foreach (SettingModelBase settingModelBase in this.Root.Settings)
			{
				xelement.Add(settingModelBase.ToXML());
			}
			return xelement;
		}

		// Token: 0x060007D0 RID: 2000 RVA: 0x00022FDC File Offset: 0x000211DC
		internal SettingModel GetSettingByGuid(string guid)
		{
			return this.Root.GetSettingByUid(guid);
		}

		// Token: 0x060007D1 RID: 2001 RVA: 0x00022FEC File Offset: 0x000211EC
		internal void ImportFrom(string filename)
		{
			string text = SiteFileHelper.Decrypt(File.ReadAllText(filename));
			this.Parse(text);
		}

		// Token: 0x060007D2 RID: 2002 RVA: 0x0002300C File Offset: 0x0002120C
		internal void ExportTo(string filename)
		{
			SiteFileHelper.SaveTo(this.ToXML(), filename);
		}
	}
}
