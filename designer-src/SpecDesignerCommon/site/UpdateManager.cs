using System;
using System.IO;
using System.Windows;
using AppLimit.NetSparkle;
using SpecDesigner.Controls.Controls;

namespace SpecDesignerCommon.Site
{
	// Token: 0x0200009C RID: 156
	public class UpdateManager
	{
		// Token: 0x170001CB RID: 459
		// (get) Token: 0x0600064C RID: 1612 RVA: 0x0001C5CC File Offset: 0x0001A7CC
		public static UpdateManager This
		{
			get
			{
				return UpdateManager._this;
			}
		}

		// Token: 0x14000018 RID: 24
		// (add) Token: 0x0600064D RID: 1613 RVA: 0x0001C5D4 File Offset: 0x0001A7D4
		// (remove) Token: 0x0600064E RID: 1614 RVA: 0x0001C60C File Offset: 0x0001A80C
		public event UpdateManager.FinishedHandler checkFinished;

		// Token: 0x170001CC RID: 460
		// (get) Token: 0x0600064F RID: 1615 RVA: 0x0001C644 File Offset: 0x0001A844
		public string Version
		{
			get
			{
				string text = this._sparkle.Version;
				if (!string.IsNullOrEmpty(text))
				{
					string[] array = text.Split(new char[] { '.' });
					text = string.Format("{0}.{1}", array[0], array[1]);
				}
				return text;
			}
		}

		// Token: 0x170001CD RID: 461
		// (get) Token: 0x06000650 RID: 1616 RVA: 0x0001C68B File Offset: 0x0001A88B
		// (set) Token: 0x06000651 RID: 1617 RVA: 0x0001C693 File Offset: 0x0001A893
		public bool IsForceUpdate
		{
			get
			{
				return this._isForceUpdate;
			}
			set
			{
				this._isForceUpdate = value;
			}
		}

		// Token: 0x170001CE RID: 462
		// (get) Token: 0x06000652 RID: 1618 RVA: 0x0001C69C File Offset: 0x0001A89C
		// (set) Token: 0x06000653 RID: 1619 RVA: 0x0001C6A4 File Offset: 0x0001A8A4
		public string AppCastIP
		{
			get
			{
				return this._appCastIP;
			}
			set
			{
				this._appCastIP = Path.Combine(value, "UpdateFeed.xml");
				if (this._sparkle != null)
				{
					this._sparkle.Dispose();
					this._sparkle.Error -= this._sparkle_Error;
					this._sparkle.Finish -= this._sparkle_Finish;
					this._sparkle.updateDetected -= this._sparkle_updateDetected;
				}
				this._sparkle = new Sparkle(this._appCastIP);
				this._sparkle.ShowDiagnosticWindow = false;
				this._sparkle.EnableSilentMode = true;
				this._sparkle.EnableServiceMode = true;
				this._sparkle.Error += this._sparkle_Error;
				this._sparkle.Finish += this._sparkle_Finish;
				if (this._isForceUpdate)
				{
					this._sparkle.updateDetected += this._sparkle_updateDetected;
					this._sparkle.StartLoop(true, true, new TimeSpan(2, 0, 0));
					return;
				}
				this._sparkle.GetVersion();
			}
		}

		// Token: 0x06000654 RID: 1620 RVA: 0x0001C7BD File Offset: 0x0001A9BD
		private void _sparkle_Error(object sender, Exception ce)
		{
			if (this.checkFinished != null)
			{
				this.checkFinished(this, new CheckVersionArgs(true, ce));
			}
		}

		// Token: 0x06000655 RID: 1621 RVA: 0x0001C7DC File Offset: 0x0001A9DC
		private void _sparkle_Finish(object sender, bool isNeedUpdate)
		{
			if (!isNeedUpdate && !this.IsForceUpdate)
			{
				DesignerMessageBox.Show((Application.Current.FindResource("Message_GetRemoteVersion") as string) + this.Version, Application.Current.FindResource("Message_GetRemoteVersion") as string);
			}
			else if (isNeedUpdate && !this.IsForceUpdate)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_RemoteHasNewVersion") as string, Application.Current.FindResource("Message_GetRemoteVersion") as string);
			}
			if (this.checkFinished != null)
			{
				this.checkFinished(this, new CheckVersionArgs(false));
			}
		}

		// Token: 0x06000656 RID: 1622 RVA: 0x0001C882 File Offset: 0x0001AA82
		private void _sparkle_updateDetected(object sender, UpdateDetectedEventArgs e)
		{
		}

		// Token: 0x06000657 RID: 1623 RVA: 0x0001C884 File Offset: 0x0001AA84
		private void Dispose()
		{
			this._sparkle.Error -= this._sparkle_Error;
			this._sparkle.Finish -= this._sparkle_Finish;
			this._sparkle.updateDetected -= this._sparkle_updateDetected;
			this._sparkle.Dispose();
		}

		// Token: 0x0400026A RID: 618
		private const string UPDATEFEED = "UpdateFeed.xml";

		// Token: 0x0400026B RID: 619
		private static UpdateManager _this = new UpdateManager();

		// Token: 0x0400026C RID: 620
		private Sparkle _sparkle;

		// Token: 0x0400026E RID: 622
		private bool _isForceUpdate = true;

		// Token: 0x0400026F RID: 623
		private string _appCastIP = string.Empty;

		// Token: 0x0200009D RID: 157
		// (Invoke) Token: 0x0600065B RID: 1627
		public delegate void FinishedHandler(object sender, CheckVersionArgs args);
	}
}
