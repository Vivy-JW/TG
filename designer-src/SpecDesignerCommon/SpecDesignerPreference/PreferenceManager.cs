using System;
using System.ComponentModel;
using System.Windows;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;

namespace SpecDesignerPreference
{
	// Token: 0x02000066 RID: 102
	public class PreferenceManager : INotifyPropertyChanged
	{
		// Token: 0x14000010 RID: 16
		// (add) Token: 0x060003EB RID: 1003 RVA: 0x00012858 File Offset: 0x00010A58
		// (remove) Token: 0x060003EC RID: 1004 RVA: 0x00012890 File Offset: 0x00010A90
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060003ED RID: 1005 RVA: 0x000128C5 File Offset: 0x00010AC5
		public void OnPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x17000106 RID: 262
		// (get) Token: 0x060003EE RID: 1006 RVA: 0x000128E1 File Offset: 0x00010AE1
		public static PreferenceManager Current
		{
			get
			{
				if (PreferenceManager._this == null)
				{
					PreferenceManager._this = new PreferenceManager();
				}
				return PreferenceManager._this;
			}
		}

		// Token: 0x060003EF RID: 1007 RVA: 0x000128F9 File Offset: 0x00010AF9
		private PreferenceManager()
		{
		}

		// Token: 0x17000107 RID: 263
		// (get) Token: 0x060003F0 RID: 1008 RVA: 0x00012901 File Offset: 0x00010B01
		public PreferenceModel Settings
		{
			get
			{
				return this._preferenceModel;
			}
		}

		// Token: 0x17000108 RID: 264
		// (get) Token: 0x060003F1 RID: 1009 RVA: 0x00012909 File Offset: 0x00010B09
		public CommonUsedElement CommonUsed
		{
			get
			{
				return this._commonUsedElement;
			}
		}

		// Token: 0x060003F2 RID: 1010 RVA: 0x00012914 File Offset: 0x00010B14
		public void LoadSettings()
		{
			if (this._isLoaded)
			{
				return;
			}
			this._isLoaded = true;
			try
			{
				this._preferenceModel = new PreferenceModel(SettingFileHelper.Open());
				this._commonUsedElement = this._preferenceModel.CommonUsedSetting;
			}
			catch (Exception ex)
			{
				DesignerMessageBox.Show(ex.Message, Application.Current.FindResource("Preference_LoadFailed") as string);
			}
		}

		// Token: 0x060003F3 RID: 1011 RVA: 0x00012988 File Offset: 0x00010B88
		public void Save()
		{
			SettingFileHelper.Save(this._preferenceModel.ToXML());
		}

		// Token: 0x060003F4 RID: 1012 RVA: 0x0001299A File Offset: 0x00010B9A
		public void Save(PreferenceModel source)
		{
			PreferenceModel.CloneSetting(source, PreferenceManager.Current.Settings);
			this.Save();
		}

		// Token: 0x060003F5 RID: 1013 RVA: 0x000129B2 File Offset: 0x00010BB2
		public void Close()
		{
		}

		// Token: 0x060003F6 RID: 1014 RVA: 0x000129B4 File Offset: 0x00010BB4
		public void Reset()
		{
			this._preferenceModel.Reset();
		}

		// Token: 0x04000181 RID: 385
		private static PreferenceManager _this;

		// Token: 0x04000182 RID: 386
		private bool _isLoaded;

		// Token: 0x04000183 RID: 387
		private PreferenceModel _preferenceModel;

		// Token: 0x04000184 RID: 388
		private CommonUsedElement _commonUsedElement;
	}
}
