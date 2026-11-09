using System;
using System.ComponentModel;
using System.Windows;
using System.Windows.Threading;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;
using SpecDesignerCommon.Exceptions;

namespace SpecDesigner.Main
{
	// Token: 0x02000020 RID: 32
	public class SpecDesignerInit
	{
		// Token: 0x14000006 RID: 6
		// (add) Token: 0x06000238 RID: 568 RVA: 0x0000A9E4 File Offset: 0x00008BE4
		// (remove) Token: 0x06000239 RID: 569 RVA: 0x0000AA1C File Offset: 0x00008C1C
		public event EventHandler Completed;

		// Token: 0x0600023A RID: 570 RVA: 0x0000AA54 File Offset: 0x00008C54
		public void Start(SpecDesignerSplashScreen specDesignerSplashScreen)
		{
			this.Dialog = specDesignerSplashScreen;
			this.worker = new BackgroundWorker();
			this.worker.DoWork += this.worker_DoWork;
			this.worker.RunWorkerCompleted += this.worker_RunWorkerCompleted;
			this.worker.RunWorkerAsync();
		}

		// Token: 0x0600023B RID: 571 RVA: 0x0000AAAC File Offset: 0x00008CAC
		private void worker_RunWorkerCompleted(object sender, RunWorkerCompletedEventArgs e)
		{
			if (this.Completed != null)
			{
				this.Completed(this, e);
			}
		}

		// Token: 0x0600023C RID: 572 RVA: 0x0000AB4C File Offset: 0x00008D4C
		private void worker_DoWork(object sender, DoWorkEventArgs e)
		{
			try
			{
				SettingManager.Get().LoadCommonData();
			}
			catch (VersionIncompatibleException ex)
			{
				VersionIncompatibleException verExp = ex;
				Application.Current.Dispatcher.Invoke(DispatcherPriority.Normal, new Action(delegate
				{
					string text = Application.Current.FindResource("Message_BasicDataRedownloading") as string;
					text = string.Format(text, verExp.Message);
					DesignerMessageBox.Show(text, Application.Current.FindResource("Message_Warning") as string);
				}));
			}
			catch (Exception)
			{
				Application.Current.Dispatcher.Invoke(DispatcherPriority.Normal, new Action(delegate
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_BeforeRedownloadingBasicData") as string, Application.Current.FindResource("Message_Warning") as string);
				}));
			}
		}

		// Token: 0x040000F4 RID: 244
		private SpecDesignerSplashScreen Dialog;

		// Token: 0x040000F5 RID: 245
		private BackgroundWorker worker;
	}
}
