using System;
using System.ComponentModel;
using SpecDesignerCommon;

namespace SpecDesigner.ViewModels
{
	// Token: 0x0200001C RID: 28
	internal class WelcomeViewModel : INotifyPropertyChanged
	{
		// Token: 0x06000221 RID: 545 RVA: 0x0000A6E3 File Offset: 0x000088E3
		public void ChangePlatform()
		{
			this.CurrentPlatformName = SettingManager.Get().CurrentSetting.Name;
		}

		// Token: 0x17000092 RID: 146
		// (get) Token: 0x06000222 RID: 546 RVA: 0x0000A6FA File Offset: 0x000088FA
		public static WelcomeViewModel This
		{
			get
			{
				return WelcomeViewModel._this;
			}
		}

		// Token: 0x17000093 RID: 147
		// (get) Token: 0x06000223 RID: 547 RVA: 0x0000A701 File Offset: 0x00008901
		// (set) Token: 0x06000224 RID: 548 RVA: 0x0000A709 File Offset: 0x00008909
		public string CurrentPlatformName
		{
			get
			{
				return this._currentPlatformName;
			}
			set
			{
				if (this._currentPlatformName != value)
				{
					this._currentPlatformName = value;
					this.OnPropertyChanged("CurrentPlatformName");
				}
			}
		}

		// Token: 0x06000225 RID: 549 RVA: 0x0000A72B File Offset: 0x0000892B
		public void ChangeShowWelcome(bool isChecked)
		{
			this.ShowWelcome = isChecked;
		}

		// Token: 0x17000094 RID: 148
		// (get) Token: 0x06000226 RID: 550 RVA: 0x0000A734 File Offset: 0x00008934
		// (set) Token: 0x06000227 RID: 551 RVA: 0x0000A73C File Offset: 0x0000893C
		public bool ShowWelcome
		{
			get
			{
				return this._showWelcome;
			}
			set
			{
				if (this._showWelcome != value)
				{
					this._showWelcome = value;
					this.OnPropertyChanged("ShowWelcome");
				}
			}
		}

		// Token: 0x14000005 RID: 5
		// (add) Token: 0x06000228 RID: 552 RVA: 0x0000A75C File Offset: 0x0000895C
		// (remove) Token: 0x06000229 RID: 553 RVA: 0x0000A794 File Offset: 0x00008994
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600022A RID: 554 RVA: 0x0000A7C9 File Offset: 0x000089C9
		public void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x040000EC RID: 236
		private static WelcomeViewModel _this = new WelcomeViewModel();

		// Token: 0x040000ED RID: 237
		public string _currentPlatformName;

		// Token: 0x040000EE RID: 238
		public bool _showWelcome;
	}
}
