using System;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x02000025 RID: 37
	public class ConnectionInfo
	{
		// Token: 0x1700003C RID: 60
		// (get) Token: 0x0600010E RID: 270 RVA: 0x00005EF3 File Offset: 0x000040F3
		public static ConnectionInfo This
		{
			get
			{
				return ConnectionInfo._this;
			}
		}

		// Token: 0x0600010F RID: 271 RVA: 0x00005EFA File Offset: 0x000040FA
		private ConnectionInfo()
		{
		}

		// Token: 0x06000110 RID: 272 RVA: 0x00005F18 File Offset: 0x00004118
		public void Initialize()
		{
			EventAggregatorManager.Global.GetEvent<RunCommandEvent>().Subscribe(new Action<string>(this.RunCommand));
		}

		// Token: 0x06000111 RID: 273 RVA: 0x00005F36 File Offset: 0x00004136
		private void RunCommand(string command)
		{
			ConnectionManager.RunProgram(command);
		}

		// Token: 0x06000112 RID: 274 RVA: 0x00005F3F File Offset: 0x0000413F
		public void CreateLogWindow()
		{
			if (this.SLW == null)
			{
				this.SLW = new ServerLog();
			}
			this.SLW.Show();
		}

		// Token: 0x06000113 RID: 275 RVA: 0x00005F6C File Offset: 0x0000416C
		private void CreateLog()
		{
			if (this.SLW == null)
			{
				this.SLW = new ServerLog();
			}
			this.SLW.Dispatcher.Invoke(new Action(delegate
			{
				this.SLW.Show();
			}), new object[0]);
		}

		// Token: 0x14000003 RID: 3
		// (add) Token: 0x06000114 RID: 276 RVA: 0x00005FA4 File Offset: 0x000041A4
		// (remove) Token: 0x06000115 RID: 277 RVA: 0x00005FDC File Offset: 0x000041DC
		public event EventHandler RequiredServerLogClose;

		// Token: 0x14000004 RID: 4
		// (add) Token: 0x06000116 RID: 278 RVA: 0x00006014 File Offset: 0x00004214
		// (remove) Token: 0x06000117 RID: 279 RVA: 0x0000604C File Offset: 0x0000424C
		public event EventHandler RequiredServerLogShow;

		// Token: 0x06000118 RID: 280 RVA: 0x00006081 File Offset: 0x00004281
		public void Close()
		{
			if (this.RequiredServerLogClose != null)
			{
				this.RequiredServerLogClose(this, EventArgs.Empty);
			}
		}

		// Token: 0x1700003D RID: 61
		// (get) Token: 0x06000119 RID: 281 RVA: 0x0000609C File Offset: 0x0000429C
		// (set) Token: 0x0600011A RID: 282 RVA: 0x000060A4 File Offset: 0x000042A4
		public bool IsLogin
		{
			get
			{
				return this._isLogin;
			}
			set
			{
				if (this._isLogin == value)
				{
					return;
				}
				this._isLogin = value;
				ConnectionStatusChangedEventArgs e = new ConnectionStatusChangedEventArgs();
				e.ServerIP = this.ServerIP;
				e.LoginName = this.LoginUser;
				e.ConnectionArea = this.Area;
				e.IsLogin = value;
				EventAggregatorManager.Global.GetEvent<ConnectionStatusChangedEvent>().Publish(e);
			}
		}

		// Token: 0x1700003E RID: 62
		// (get) Token: 0x0600011B RID: 283 RVA: 0x00006103 File Offset: 0x00004303
		// (set) Token: 0x0600011C RID: 284 RVA: 0x00006119 File Offset: 0x00004319
		public string LoginUser
		{
			get
			{
				return SettingManager.Get().CurrentSetting.Connection.Login;
			}
			set
			{
				SettingManager.Get().CurrentSetting.Connection.Login = value;
			}
		}

		// Token: 0x1700003F RID: 63
		// (get) Token: 0x0600011D RID: 285 RVA: 0x00006130 File Offset: 0x00004330
		// (set) Token: 0x0600011E RID: 286 RVA: 0x00006146 File Offset: 0x00004346
		public string ServerIP
		{
			get
			{
				return SettingManager.Get().CurrentSetting.Connection.IP;
			}
			set
			{
				SettingManager.Get().CurrentSetting.Connection.IP = value;
			}
		}

		// Token: 0x17000040 RID: 64
		// (get) Token: 0x0600011F RID: 287 RVA: 0x0000615D File Offset: 0x0000435D
		// (set) Token: 0x06000120 RID: 288 RVA: 0x00006173 File Offset: 0x00004373
		public string Area
		{
			get
			{
				return SettingManager.Get().CurrentSetting.Connection.Area;
			}
			set
			{
				SettingManager.Get().CurrentSetting.Connection.Area = value;
			}
		}

		// Token: 0x0400006F RID: 111
		private static ConnectionInfo _this = new ConnectionInfo
		{
			IsLogin = false
		};

		// Token: 0x04000072 RID: 114
		private bool _isLogin;

		// Token: 0x04000073 RID: 115
		public string ProgramName = "";

		// Token: 0x04000074 RID: 116
		public string UploadType = "";

		// Token: 0x04000075 RID: 117
		public ITTNetwork Telnet;

		// Token: 0x04000076 RID: 118
		public ServerLog SLW;
	}
}
