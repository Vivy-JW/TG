using System;
using System.Linq;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.Site.Behaviors;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x02000081 RID: 129
	public class SettingModel : SettingModelBase, IDragable, IDropable
	{
		// Token: 0x1700015F RID: 351
		// (get) Token: 0x060004F5 RID: 1269 RVA: 0x00016DBE File Offset: 0x00014FBE
		public ConnectionSetting Connection
		{
			get
			{
				return this._connect;
			}
		}

		// Token: 0x17000160 RID: 352
		// (get) Token: 0x060004F6 RID: 1270 RVA: 0x00016DC6 File Offset: 0x00014FC6
		// (set) Token: 0x060004F7 RID: 1271 RVA: 0x00016DCE File Offset: 0x00014FCE
		public string UID { get; internal set; }

		// Token: 0x060004F8 RID: 1272 RVA: 0x00016DD7 File Offset: 0x00014FD7
		public SettingModel()
		{
		}

		// Token: 0x060004F9 RID: 1273 RVA: 0x00016DDF File Offset: 0x00014FDF
		public SettingModel(XElement element)
		{
			this._connect = new ConnectionSetting(this);
			this.ParseAttributes(element);
		}

		// Token: 0x060004FA RID: 1274 RVA: 0x00016DFC File Offset: 0x00014FFC
		private void ParseAttributes(XElement element)
		{
			XAttribute xattribute = element.Attribute("name");
			base.Name = ((xattribute == null) ? "" : xattribute.Value);
			this.UID = ((element.Attribute("uid") == null) ? Guid.NewGuid().ToString() : element.Attribute("uid").Value);
			base.Order = SettingModelBase.ParseOrder(element);
			this.Connection.Parse(element.Element("Connection"));
		}

		// Token: 0x060004FB RID: 1275 RVA: 0x00016E9C File Offset: 0x0001509C
		public static SettingModel Create()
		{
			return new SettingModel(SettingModel.GetXMLStructure(false, null));
		}

		// Token: 0x060004FC RID: 1276 RVA: 0x00016EB7 File Offset: 0x000150B7
		public override SettingModelBase Clone()
		{
			return new SettingModel(SettingModel.GetXMLStructure(true, this));
		}

		// Token: 0x060004FD RID: 1277 RVA: 0x00016EC5 File Offset: 0x000150C5
		public override XElement ToXML()
		{
			return SettingModel.GetXMLStructure(true, this);
		}

		// Token: 0x17000161 RID: 353
		// (get) Token: 0x060004FE RID: 1278 RVA: 0x00016ECE File Offset: 0x000150CE
		// (set) Token: 0x060004FF RID: 1279 RVA: 0x00016ED6 File Offset: 0x000150D6
		public bool IsCheckedRemoteVersion { get; protected internal set; }

		// Token: 0x06000500 RID: 1280 RVA: 0x00016EE0 File Offset: 0x000150E0
		private static XElement GetXMLStructure(bool copy, SettingModel model)
		{
			int num = ((model == null) ? (-1) : model.Order);
			if (model != null && -1 == model.Order)
			{
				SettingModelBase settingModelBase = model.Parent.Settings.LastOrDefault<SettingModelBase>();
				if (settingModelBase != null)
				{
					num = settingModelBase.Order + 1;
				}
			}
			return new XElement("Setting", new object[]
			{
				new XAttribute("name", copy ? model.Name : "New"),
				new XAttribute("uid", copy ? model.UID : Guid.NewGuid().ToString()),
				new XAttribute("order", num),
				new XElement("Connection", new object[]
				{
					new XElement("Version", copy ? model.Connection.Version : ""),
					new XElement("Protocol", copy ? model.Connection.Protocol : "ssh"),
					new XElement("IP", copy ? model.Connection.IP : ""),
					new XElement("Port", copy ? model.Connection.Port : "22"),
					new XElement("Login", copy ? model.Connection.Login : ""),
					new XElement("Password", copy ? model.Connection.Password : ""),
					new XElement("Area", copy ? model.Connection.Area : "1"),
					new XElement("Workspace", copy ? model.Connection.Workspace : "C:\\TT"),
					new XElement("LoginPrompt", copy ? model.Connection.LoginPrompt : "Login:"),
					new XElement("PasswordPrompt", copy ? model.Connection.PasswordPrompt : "Password:"),
					new XElement("AreaPrompt", copy ? model.Connection.AreaPrompt : "(*)Exit"),
					new XElement("UpdateURL", copy ? model.Connection.UpdateURL : ""),
					new XElement("Zone", copy ? model.Connection.Zone : "topprd"),
					new XElement("ExtraCommand", copy ? model.Connection.ExtraCommand : string.Empty)
				})
			});
		}

		// Token: 0x06000501 RID: 1281 RVA: 0x000171F9 File Offset: 0x000153F9
		public override SettingModel GetSettingByUid(string uid)
		{
			if (this.UID == uid)
			{
				return this;
			}
			return null;
		}

		// Token: 0x17000162 RID: 354
		// (get) Token: 0x06000502 RID: 1282 RVA: 0x0001720C File Offset: 0x0001540C
		public Type DragType
		{
			get
			{
				return typeof(SettingModelBase);
			}
		}

		// Token: 0x17000163 RID: 355
		// (get) Token: 0x06000503 RID: 1283 RVA: 0x00017218 File Offset: 0x00015418
		public Type AllowType
		{
			get
			{
				return typeof(SettingModelBase);
			}
		}

		// Token: 0x06000504 RID: 1284 RVA: 0x00017224 File Offset: 0x00015424
		public void DropOver(DragEventArgs dragEvetnArgs)
		{
		}

		// Token: 0x06000505 RID: 1285 RVA: 0x00017228 File Offset: 0x00015428
		public void Drop(IDragable drag)
		{
			SettingModelBase settingModelBase = drag as SettingModelBase;
			base.Parent.Insert(settingModelBase, base.Order);
		}

		// Token: 0x06000506 RID: 1286 RVA: 0x0001724E File Offset: 0x0001544E
		public bool CanDrop(IDragable drag)
		{
			return base.Parent != null;
		}

		// Token: 0x040001DE RID: 478
		private ConnectionSetting _connect;
	}
}
