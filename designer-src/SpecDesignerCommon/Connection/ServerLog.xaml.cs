using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Text;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Threading;
using SpecDesigner.Controls.Controls;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x02000050 RID: 80
	public partial class ServerLog : Window
	{
		// Token: 0x060002A4 RID: 676 RVA: 0x0000BAAC File Offset: 0x00009CAC
		public ServerLog()
		{
			this.InitializeComponent();
			base.CommandBindings.Add(new CommandBinding(ServerLogCommands.SaveCommand, new ExecutedRoutedEventHandler(this.OnSave), new CanExecuteRoutedEventHandler(this.CanSave)));
			base.CommandBindings.Add(new CommandBinding(ServerLogCommands.ClearCommand, new ExecutedRoutedEventHandler(this.OnClear), new CanExecuteRoutedEventHandler(this.CanClear)));
			this.tb_ServerLog.TextChanged += this.tb_ServerLog_TextChanged;
			this.logData = new LogData();
			base.DataContext = this.logData;
		}

		// Token: 0x060002A5 RID: 677 RVA: 0x0000BB73 File Offset: 0x00009D73
		private void tb_ServerLog_TextChanged(object sender, TextChangedEventArgs e)
		{
			if (!FocusManager.GetIsFocusScope(this.tb_ServerLog))
			{
				this.tb_ServerLog.ScrollToEnd();
			}
		}

		// Token: 0x060002A6 RID: 678 RVA: 0x0000BB8D File Offset: 0x00009D8D
		protected override void OnClosing(CancelEventArgs e)
		{
			if (this._isClose)
			{
				return;
			}
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
		}

		// Token: 0x060002A7 RID: 679 RVA: 0x0000BBA8 File Offset: 0x00009DA8
		public new void Show()
		{
			this.StartListening();
			base.UpdateLayout();
			if (base.Visibility == Visibility.Visible)
			{
				return;
			}
			if (WindowState.Minimized == base.WindowState)
			{
				base.WindowState = WindowState.Normal;
			}
			base.Activate();
			base.Topmost = true;
			base.Topmost = false;
			base.Focus();
			base.Visibility = Visibility.Visible;
			Dispatcher.Run();
		}

		// Token: 0x060002A8 RID: 680 RVA: 0x0000BC02 File Offset: 0x00009E02
		private void RequiredClosed(object sender, EventArgs e)
		{
			this.StopListening();
			base.Dispatcher.InvokeShutdown();
		}

		// Token: 0x060002A9 RID: 681 RVA: 0x0000BC18 File Offset: 0x00009E18
		public void StartListening()
		{
			if (ConnectionInfo.This.Telnet != null)
			{
				ConnectionInfo.This.Telnet.OnDataReceived -= this.Telnet_OnDataReceived;
				ConnectionInfo.This.Telnet.OnDataReceived += this.Telnet_OnDataReceived;
			}
		}

		// Token: 0x060002AA RID: 682 RVA: 0x0000BC67 File Offset: 0x00009E67
		private void Telnet_OnDataReceived(object sender, DataReceivedEventArgs args)
		{
			this.logData.Append(args.Data);
		}

		// Token: 0x060002AB RID: 683 RVA: 0x0000BC7A File Offset: 0x00009E7A
		public void StopListening()
		{
			if (ConnectionInfo.This.Telnet != null)
			{
				ConnectionInfo.This.Telnet.OnDataReceived -= this.Telnet_OnDataReceived;
			}
		}

		// Token: 0x060002AC RID: 684 RVA: 0x0000BCA3 File Offset: 0x00009EA3
		private void CanSave(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = this.logData.Data.Length > 0;
		}

		// Token: 0x060002AD RID: 685 RVA: 0x0000BCC4 File Offset: 0x00009EC4
		private void OnSave(object sender, ExecutedRoutedEventArgs e)
		{
			if (!Directory.Exists(this.LOGPATH))
			{
				Directory.CreateDirectory(this.LOGPATH);
			}
			string text = string.Format("{0}serverLog_{1}.log", this.LOGPATH, DateTime.Now.ToString("yyyyMMdd"));
			using (StreamWriter streamWriter = new StreamWriter(text, true, Encoding.UTF8))
			{
				streamWriter.WriteLine(this.tb_ServerLog.Text);
			}
			string text2 = Application.Current.FindResource("Message_Saved") as string;
			DesignerMessageBox.Show(string.Format(text2, text), "Server Log", MessageBoxButton.OK, MessageBoxImage.Asterisk);
		}

		// Token: 0x060002AE RID: 686 RVA: 0x0000BD74 File Offset: 0x00009F74
		private void CanClear(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = this.logData.Data.Length > 0;
		}

		// Token: 0x060002AF RID: 687 RVA: 0x0000BD93 File Offset: 0x00009F93
		private void OnClear(object sender, ExecutedRoutedEventArgs e)
		{
			this.logData.Clear();
		}

		// Token: 0x040000FF RID: 255
		private readonly string LOGPATH = string.Format("{0}\\log\\", SettingManager.Get().CurrentSetting.Connection.Workspace);

		// Token: 0x04000100 RID: 256
		private bool _isClose;

		// Token: 0x04000101 RID: 257
		private LogData logData;
	}
}
