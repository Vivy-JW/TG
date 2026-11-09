using System;
using System.ComponentModel;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x020000AF RID: 175
	public class TOPSTDSettingModel : INotifyPropertyChanged
	{
		// Token: 0x1700021A RID: 538
		// (get) Token: 0x0600075D RID: 1885 RVA: 0x00020FFA File Offset: 0x0001F1FA
		public string Login
		{
			get
			{
				return "topstd";
			}
		}

		// Token: 0x1700021B RID: 539
		// (get) Token: 0x0600075E RID: 1886 RVA: 0x00021001 File Offset: 0x0001F201
		// (set) Token: 0x0600075F RID: 1887 RVA: 0x00021009 File Offset: 0x0001F209
		public string Password
		{
			get
			{
				return this._password;
			}
			set
			{
				this._password = value;
				this.NotifyPropertyChanged("Password");
			}
		}

		// Token: 0x1700021C RID: 540
		// (get) Token: 0x06000760 RID: 1888 RVA: 0x0002101D File Offset: 0x0001F21D
		// (set) Token: 0x06000761 RID: 1889 RVA: 0x00021025 File Offset: 0x0001F225
		public bool IsLogin
		{
			get
			{
				return this._isLogin;
			}
			set
			{
				this._isLogin = value;
				this.NotifyPropertyChanged("IsLogin");
			}
		}

		// Token: 0x1400001B RID: 27
		// (add) Token: 0x06000762 RID: 1890 RVA: 0x0002103C File Offset: 0x0001F23C
		// (remove) Token: 0x06000763 RID: 1891 RVA: 0x00021074 File Offset: 0x0001F274
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000764 RID: 1892 RVA: 0x000210A9 File Offset: 0x0001F2A9
		internal void NotifyPropertyChanged(string info)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(info));
			}
		}

		// Token: 0x040002A2 RID: 674
		private string _password = string.Empty;

		// Token: 0x040002A3 RID: 675
		private bool _isLogin;
	}
}
