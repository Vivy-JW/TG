using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.Connection;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x02000082 RID: 130
	public class ConnectionSetting : INotifyPropertyChanged, IDataErrorInfo
	{
		// Token: 0x06000507 RID: 1287 RVA: 0x0001725C File Offset: 0x0001545C
		public ConnectionSetting(string userName, string password)
		{
			this.Login = userName;
			this.Password = password;
		}

		// Token: 0x06000508 RID: 1288 RVA: 0x000172BC File Offset: 0x000154BC
		public ConnectionSetting(SettingModel settingModel)
		{
			this.Setting = settingModel;
		}

		// Token: 0x06000509 RID: 1289 RVA: 0x00017315 File Offset: 0x00015515
		public void SetRemoveVersionChecked(bool check)
		{
			this.Setting.IsCheckedRemoteVersion = check;
		}

		// Token: 0x17000164 RID: 356
		// (get) Token: 0x0600050A RID: 1290 RVA: 0x00017323 File Offset: 0x00015523
		public string[] Protocols
		{
			get
			{
				return this._protocols;
			}
		}

		// Token: 0x17000165 RID: 357
		// (get) Token: 0x0600050B RID: 1291 RVA: 0x0001732B File Offset: 0x0001552B
		// (set) Token: 0x0600050C RID: 1292 RVA: 0x00017334 File Offset: 0x00015534
		public string Version
		{
			get
			{
				return this._version;
			}
			set
			{
				this._version = value;
				try
				{
					RegistryReader.GetExeDir(value);
				}
				catch
				{
				}
				this.OnPropertyChanged("Version");
				this.OnPropertyChanged("IsInstalled");
			}
		}

		// Token: 0x17000166 RID: 358
		// (get) Token: 0x0600050D RID: 1293 RVA: 0x0001737C File Offset: 0x0001557C
		// (set) Token: 0x0600050E RID: 1294 RVA: 0x00017384 File Offset: 0x00015584
		public string Protocol
		{
			get
			{
				return this._protocol;
			}
			set
			{
				if (value != null)
				{
					if (!(value == "ssh"))
					{
						if (value == "telnet")
						{
							this._port = "23";
						}
					}
					else
					{
						this._port = "22";
					}
				}
				if (this._protocol != value)
				{
					this.CloseConnection();
					this._protocol = value;
					this.OnPropertyChanged("Port");
					this.OnPropertyChanged("Protocol");
				}
			}
		}

		// Token: 0x17000167 RID: 359
		// (get) Token: 0x0600050F RID: 1295 RVA: 0x000173FB File Offset: 0x000155FB
		// (set) Token: 0x06000510 RID: 1296 RVA: 0x00017404 File Offset: 0x00015604
		public string Port
		{
			get
			{
				return this._port;
			}
			set
			{
				string protocol;
				if (string.IsNullOrEmpty(value) && (protocol = this.Protocol) != null)
				{
					if (!(protocol == "telnet"))
					{
						if (protocol == "ssh")
						{
							value = "22";
						}
					}
					else
					{
						value = "23";
					}
				}
				this._port = value;
				this.OnPropertyChanged("Port");
			}
		}

		// Token: 0x17000168 RID: 360
		// (get) Token: 0x06000511 RID: 1297 RVA: 0x00017461 File Offset: 0x00015661
		// (set) Token: 0x06000512 RID: 1298 RVA: 0x0001746C File Offset: 0x0001566C
		public string IP
		{
			get
			{
				return this._ip;
			}
			set
			{
				string text = value.Trim();
				if (this._ip != text)
				{
					this.CloseConnection();
					this._ip = text;
					if (text != "")
					{
						this._updateurl = string.Empty;
						this.OnPropertyChanged("UpdateURL");
					}
					this.OnPropertyChanged("IP");
				}
			}
		}

		// Token: 0x17000169 RID: 361
		// (get) Token: 0x06000513 RID: 1299 RVA: 0x000174C9 File Offset: 0x000156C9
		// (set) Token: 0x06000514 RID: 1300 RVA: 0x000174D1 File Offset: 0x000156D1
		public string Login
		{
			get
			{
				return this._login;
			}
			set
			{
				if (this._login != value)
				{
					this.CloseConnection();
					this._login = value;
					this.OnPropertyChanged("Login");
				}
			}
		}

		// Token: 0x1700016A RID: 362
		// (get) Token: 0x06000515 RID: 1301 RVA: 0x000174F9 File Offset: 0x000156F9
		// (set) Token: 0x06000516 RID: 1302 RVA: 0x00017501 File Offset: 0x00015701
		public string Password
		{
			get
			{
				return this._password;
			}
			set
			{
				if (this._password != value)
				{
					this.CloseConnection();
					this._password = value;
					this.OnPropertyChanged("Password");
				}
			}
		}

		// Token: 0x1700016B RID: 363
		// (get) Token: 0x06000517 RID: 1303 RVA: 0x00017529 File Offset: 0x00015729
		// (set) Token: 0x06000518 RID: 1304 RVA: 0x00017531 File Offset: 0x00015731
		public string Area
		{
			get
			{
				return this._area;
			}
			set
			{
				if (this._area != value)
				{
					this.CloseConnection();
					this._area = value;
					this.OnPropertyChanged("Area");
				}
			}
		}

		// Token: 0x1700016C RID: 364
		// (get) Token: 0x06000519 RID: 1305 RVA: 0x00017559 File Offset: 0x00015759
		// (set) Token: 0x0600051A RID: 1306 RVA: 0x00017561 File Offset: 0x00015761
		public string Zone
		{
			get
			{
				return this._zone;
			}
			set
			{
				if (this._zone != value)
				{
					this.CloseConnection();
					this._zone = value;
					this.OnPropertyChanged("Zone");
					this._updateurl = string.Empty;
				}
			}
		}

		// Token: 0x1700016D RID: 365
		// (get) Token: 0x0600051B RID: 1307 RVA: 0x00017594 File Offset: 0x00015794
		// (set) Token: 0x0600051C RID: 1308 RVA: 0x0001759C File Offset: 0x0001579C
		public string Workspace
		{
			get
			{
				return this._workspace;
			}
			set
			{
				if (this._workspace == value)
				{
					return;
				}
				this._workspace = value;
				if (this._workspace != "" && !this._workspace.EndsWith(Path.DirectorySeparatorChar.ToString()))
				{
					this._workspace += Path.DirectorySeparatorChar;
				}
				this.OnPropertyChanged("Workspace");
			}
		}

		// Token: 0x1700016E RID: 366
		// (get) Token: 0x0600051D RID: 1309 RVA: 0x00017611 File Offset: 0x00015811
		// (set) Token: 0x0600051E RID: 1310 RVA: 0x00017619 File Offset: 0x00015819
		public string LoginPrompt
		{
			get
			{
				return this._loginprompt;
			}
			set
			{
				if (this._loginprompt != value)
				{
					this.CloseConnection();
					this._loginprompt = value;
					this.OnPropertyChanged("LoginPrompt");
				}
			}
		}

		// Token: 0x1700016F RID: 367
		// (get) Token: 0x0600051F RID: 1311 RVA: 0x00017641 File Offset: 0x00015841
		// (set) Token: 0x06000520 RID: 1312 RVA: 0x00017649 File Offset: 0x00015849
		public string PasswordPrompt
		{
			get
			{
				return this._passwordprompt;
			}
			set
			{
				if (this._passwordprompt != value)
				{
					this.CloseConnection();
					this._passwordprompt = value;
					this.OnPropertyChanged("PasswordPrompt");
				}
			}
		}

		// Token: 0x17000170 RID: 368
		// (get) Token: 0x06000521 RID: 1313 RVA: 0x00017671 File Offset: 0x00015871
		// (set) Token: 0x06000522 RID: 1314 RVA: 0x00017679 File Offset: 0x00015879
		public string AreaPrompt
		{
			get
			{
				return this._areaprompt;
			}
			set
			{
				if (this._areaprompt != value)
				{
					this.CloseConnection();
					this._areaprompt = value;
					this.OnPropertyChanged("AreaPrompt");
				}
			}
		}

		// Token: 0x17000171 RID: 369
		// (get) Token: 0x06000523 RID: 1315 RVA: 0x000176A4 File Offset: 0x000158A4
		// (set) Token: 0x06000524 RID: 1316 RVA: 0x00017747 File Offset: 0x00015947
		public string UpdateURL
		{
			get
			{
				if (string.IsNullOrEmpty(this.IP))
				{
					return string.Empty;
				}
				if (string.IsNullOrEmpty(this._updateurl))
				{
					Uri uri = new Uri(string.Format("http://{0}", this.IP)).Append(new string[] { this.Zone, "clitool", "designer" });
					this._updateurl = uri.AbsoluteUri;
				}
				if (!this._updateurl.EndsWith("/"))
				{
					this._updateurl += "/";
				}
				return this._updateurl;
			}
			set
			{
				if (!string.IsNullOrEmpty(value) && !value.EndsWith("/"))
				{
					value += "/";
				}
				this._updateurl = value;
				this.OnPropertyChanged("UpdateURL");
			}
		}

		// Token: 0x17000172 RID: 370
		// (get) Token: 0x06000525 RID: 1317 RVA: 0x00017780 File Offset: 0x00015980
		public string UserGuideURL
		{
			get
			{
				if (string.IsNullOrEmpty(this.IP))
				{
					return string.Empty;
				}
				if (string.IsNullOrEmpty(this._userGuideurl))
				{
					Uri uri = new Uri(string.Format("http://{0}", this.IP)).Append(new string[] { this.Zone, "clitool", "doc" });
					this._userGuideurl = uri.AbsoluteUri;
				}
				return this._userGuideurl;
			}
		}

		// Token: 0x17000173 RID: 371
		// (get) Token: 0x06000526 RID: 1318 RVA: 0x000177FB File Offset: 0x000159FB
		// (set) Token: 0x06000527 RID: 1319 RVA: 0x00017803 File Offset: 0x00015A03
		public string ExtraCommand
		{
			get
			{
				return this._extraCommand;
			}
			set
			{
				if (this._extraCommand != value)
				{
					this.CloseConnection();
					this._extraCommand = value;
					this.OnPropertyChanged("ExtraCommand");
				}
			}
		}

		// Token: 0x17000174 RID: 372
		// (get) Token: 0x06000528 RID: 1320 RVA: 0x0001782C File Offset: 0x00015A2C
		public bool IsInstalled
		{
			get
			{
				bool flag;
				try
				{
					string text = string.Empty;
					try
					{
						text = RegistryReader.GetExeDir(this.Version);
					}
					catch
					{
					}
					flag = !string.IsNullOrEmpty(text);
				}
				catch
				{
					flag = false;
				}
				return flag;
			}
		}

		// Token: 0x06000529 RID: 1321 RVA: 0x00017880 File Offset: 0x00015A80
		internal void Parse(XElement element)
		{
			this._version = ((element.Element("Version") == null) ? null : element.Element("Version").Value);
			this._ip = ((element.Element("IP") == null) ? null : element.Element("IP").Value);
			this._login = ((element.Element("Login") == null) ? null : element.Element("Login").Value);
			this._password = ((element.Element("Password") == null) ? null : element.Element("Password").Value);
			this._area = ((element.Element("Area") == null) ? null : element.Element("Area").Value);
			this._workspace = ((element.Element("Workspace") == null) ? null : element.Element("Workspace").Value);
			this._loginprompt = ((element.Element("LoginPrompt") == null) ? null : element.Element("LoginPrompt").Value);
			this._passwordprompt = ((element.Element("PasswordPrompt") == null) ? null : element.Element("PasswordPrompt").Value);
			this._areaprompt = ((element.Element("AreaPrompt") == null) ? null : element.Element("AreaPrompt").Value);
			this._updateurl = ((element.Element("UpdateURL") == null) ? null : element.Element("UpdateURL").Value);
			if (this._updateurl.Split(new char[] { '/' }).Count<string>() < 7)
			{
				this._updateurl = string.Empty;
			}
			this._zone = ((element.Element("Zone") == null) ? null : element.Element("Zone").Value);
			if (!string.IsNullOrEmpty(this._updateurl) && string.IsNullOrEmpty(this._zone) && Regex.IsMatch(this._updateurl, ".*/(?<zone>[^/]*)/clitool/designer/"))
			{
				Match match = Regex.Match(this._updateurl, ".*/(?<zone>[^/]*)/clitool/designer/", RegexOptions.IgnoreCase);
				this._zone = match.Groups["zone"].Value;
			}
			this._protocol = ((element.Element("Protocol") == null) ? null : element.Element("Protocol").Value);
			this._port = ((element.Element("Port") == null) ? null : element.Element("Port").Value);
			this._extraCommand = ((element.Element("ExtraCommand") == null) ? string.Empty : element.Element("ExtraCommand").Value);
		}

		// Token: 0x0600052A RID: 1322 RVA: 0x00017BB8 File Offset: 0x00015DB8
		public override int GetHashCode()
		{
			return this.IP.GetHashCode() ^ this.Login.GetHashCode() ^ this.Password.GetHashCode() ^ this.Area.GetHashCode() ^ this.Workspace.GetHashCode() ^ this.LoginPrompt.GetHashCode() ^ this.PasswordPrompt.GetHashCode() ^ this.AreaPrompt.GetHashCode();
		}

		// Token: 0x0600052B RID: 1323 RVA: 0x00017C24 File Offset: 0x00015E24
		public override bool Equals(object obj)
		{
			ConnectionSetting connectionSetting = obj as ConnectionSetting;
			return obj != null && connectionSetting == this;
		}

		// Token: 0x0600052C RID: 1324 RVA: 0x00017C44 File Offset: 0x00015E44
		public static bool operator ==(ConnectionSetting conn1, ConnectionSetting conn2)
		{
			return object.ReferenceEquals(conn1, conn2) || (conn1 != null && conn2 != null && (conn1.IP == conn2.IP && conn1.Login == conn2.Login && conn1.Password == conn2.Password && conn1.Area == conn2.Area && conn1.Workspace == conn2.Workspace && conn1.LoginPrompt == conn2.LoginPrompt && conn1.PasswordPrompt == conn2.PasswordPrompt && conn1.AreaPrompt == conn2.AreaPrompt));
		}

		// Token: 0x0600052D RID: 1325 RVA: 0x00017D02 File Offset: 0x00015F02
		public void Save()
		{
			SiteManager.ConfirmCommand.Execute(null);
		}

		// Token: 0x0600052E RID: 1326 RVA: 0x00017D0F File Offset: 0x00015F0F
		public static bool operator !=(ConnectionSetting conn1, ConnectionSetting conn2)
		{
			return !(conn1 == conn2);
		}

		// Token: 0x14000014 RID: 20
		// (add) Token: 0x0600052F RID: 1327 RVA: 0x00017D1C File Offset: 0x00015F1C
		// (remove) Token: 0x06000530 RID: 1328 RVA: 0x00017D54 File Offset: 0x00015F54
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000531 RID: 1329 RVA: 0x00017D89 File Offset: 0x00015F89
		public void OnPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x17000175 RID: 373
		// (get) Token: 0x06000532 RID: 1330 RVA: 0x00017DA5 File Offset: 0x00015FA5
		public string Error
		{
			get
			{
				return this.error;
			}
		}

		// Token: 0x17000176 RID: 374
		public string this[string columnName]
		{
			get
			{
				this.error = null;
				if (columnName != null)
				{
					if (!(columnName == "Version"))
					{
						if (!(columnName == "IP"))
						{
							if (!(columnName == "Zone"))
							{
								if (!(columnName == "Area"))
								{
									if (columnName == "Workspace")
									{
										List<string> list = new List<string>();
										if (string.IsNullOrEmpty(this.Workspace))
										{
											list.Add(Application.Current.FindResource("Site_workspaceIsRequired") as string);
										}
										if (!Directory.Exists(this.Workspace))
										{
											list.Add(Application.Current.FindResource("Site_WorkspaceIsNotExist") as string);
										}
										if (this.IsUsed(this.Workspace, this.Setting.UID))
										{
											list.Add(string.Format(Application.Current.FindResource("Site_WorkspaceIsUsed") as string, ""));
										}
										if (Regex.IsMatch(this.Workspace.Replace(Path.GetPathRoot(this.Workspace), ""), "[^a-zA-Z0-9_\\-\\\\]"))
										{
											list.Add(Application.Current.FindResource("Message_PathFormatIncorrent") as string);
										}
										this.error = ((list.Count<string>() > 0) ? string.Join("\n", list) : null);
									}
								}
								else if (string.IsNullOrEmpty(this.Area))
								{
									this.error = Application.Current.FindResource("Site_AreaIsRequired") as string;
								}
							}
							else if (string.IsNullOrEmpty(this.Zone))
							{
								this.error = Application.Current.FindResource("Site_ZoneIsRequired") as string;
							}
						}
						else if (string.IsNullOrEmpty(this.IP))
						{
							this.error = Application.Current.FindResource("Site_IPAddressIsRequired") as string;
						}
						else if (!Regex.IsMatch(this.IP, "^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$") || !Regex.IsMatch(this.IP, "^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$"))
						{
							this.error = "格式不正確，請輸入IP或DNS名稱";
						}
					}
					else if (string.IsNullOrEmpty(this.Version))
					{
						this.error = Application.Current.FindResource("Site_VersionNotChecked") as string;
					}
				}
				return this.error;
			}
		}

		// Token: 0x17000177 RID: 375
		// (get) Token: 0x06000534 RID: 1332 RVA: 0x00018000 File Offset: 0x00016200
		public bool IsValidate
		{
			get
			{
				this.error = this["IP"];
				if (this.error == null)
				{
					this.error = this["Workspace"];
				}
				if (this.error == null)
				{
					this.error = this["Version"];
				}
				return this.error == null;
			}
		}

		// Token: 0x06000535 RID: 1333 RVA: 0x0001805C File Offset: 0x0001625C
		public bool IsUsed(string dir, string uid)
		{
			SettingModelBase root = this.GetRoot();
			return root.CheckWorkspaceIsUsed(dir, uid);
		}

		// Token: 0x06000536 RID: 1334 RVA: 0x00018078 File Offset: 0x00016278
		private SettingModelBase GetRoot()
		{
			SettingFolderModel settingFolderModel = this.Setting.Parent;
			if (settingFolderModel == null)
			{
				return this.Setting;
			}
			if (settingFolderModel.Parent == null)
			{
				return settingFolderModel;
			}
			while ((settingFolderModel = settingFolderModel.Parent) != null)
			{
				if (settingFolderModel.Parent == null)
				{
					return settingFolderModel;
				}
			}
			return null;
		}

		// Token: 0x06000537 RID: 1335 RVA: 0x000180BC File Offset: 0x000162BC
		public bool InWorkspace(string tzpDir)
		{
			if (string.IsNullOrEmpty(this._workspace))
			{
				return false;
			}
			if (!tzpDir.EndsWith(Path.DirectorySeparatorChar.ToString()))
			{
				tzpDir += Path.DirectorySeparatorChar.ToString();
			}
			if (!this._workspace.EndsWith(Path.DirectorySeparatorChar.ToString()))
			{
				this._workspace += Path.DirectorySeparatorChar.ToString();
			}
			return tzpDir.StartsWith(this._workspace, StringComparison.CurrentCultureIgnoreCase);
		}

		// Token: 0x06000538 RID: 1336 RVA: 0x00018147 File Offset: 0x00016347
		private void CloseConnection()
		{
			ConnectionManager.CloseTelnetChannel();
		}

		// Token: 0x040001E1 RID: 481
		private const string ValidIpAddressRegex = "^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$";

		// Token: 0x040001E2 RID: 482
		private const string ValidHostnameRegex = "^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$";

		// Token: 0x040001E3 RID: 483
		private string _version;

		// Token: 0x040001E4 RID: 484
		private string _protocol = "ssh";

		// Token: 0x040001E5 RID: 485
		private string _port = "23";

		// Token: 0x040001E6 RID: 486
		private string _ip;

		// Token: 0x040001E7 RID: 487
		private string _login;

		// Token: 0x040001E8 RID: 488
		private string _password;

		// Token: 0x040001E9 RID: 489
		private string _zone;

		// Token: 0x040001EA RID: 490
		private string _area;

		// Token: 0x040001EB RID: 491
		private string _workspace;

		// Token: 0x040001EC RID: 492
		private string _loginprompt;

		// Token: 0x040001ED RID: 493
		private string _passwordprompt;

		// Token: 0x040001EE RID: 494
		private string _areaprompt;

		// Token: 0x040001EF RID: 495
		private string _updateurl;

		// Token: 0x040001F0 RID: 496
		private string _userGuideurl;

		// Token: 0x040001F1 RID: 497
		private string _extraCommand = string.Empty;

		// Token: 0x040001F2 RID: 498
		private string[] _protocols = new string[] { "telnet", "ssh" };

		// Token: 0x040001F3 RID: 499
		public SettingModel Setting;

		// Token: 0x040001F5 RID: 501
		private string error;
	}
}
