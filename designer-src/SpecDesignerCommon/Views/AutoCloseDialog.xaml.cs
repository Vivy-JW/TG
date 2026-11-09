using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Timers;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesignerCommon.Views
{
	// Token: 0x02000026 RID: 38
	public partial class AutoCloseDialog : Window, IDisposable
	{
		// Token: 0x17000041 RID: 65
		// (get) Token: 0x06000123 RID: 291 RVA: 0x000061AC File Offset: 0x000043AC
		// (set) Token: 0x06000124 RID: 292 RVA: 0x000061BE File Offset: 0x000043BE
		public string Message
		{
			get
			{
				return (string)base.GetValue(AutoCloseDialog.MessageProperty);
			}
			set
			{
				base.SetValue(AutoCloseDialog.MessageProperty, value);
			}
		}

		// Token: 0x06000125 RID: 293 RVA: 0x000061CC File Offset: 0x000043CC
		public static void OnMessageChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			AutoCloseDialog autoCloseDialog = sender as AutoCloseDialog;
			if (autoCloseDialog != null)
			{
				autoCloseDialog.messageLabel.Content = e.NewValue;
			}
		}

		// Token: 0x06000126 RID: 294 RVA: 0x000061F8 File Offset: 0x000043F8
		public AutoCloseDialog()
		{
			this.InitializeComponent();
			this._startTime = DateTime.Now;
			this._timer = new Timer(this._timeSpan.TotalMilliseconds);
			this._timer.Elapsed += this._timer_Elapsed;
			this._timer.Interval = 1000.0;
			this._timer.Start();
		}

		// Token: 0x06000127 RID: 295 RVA: 0x000062E4 File Offset: 0x000044E4
		private void _timer_Elapsed(object sender, ElapsedEventArgs e)
		{
			int interval = (int)(DateTime.Now - this._startTime).TotalSeconds;
			if (interval < this._timeSpan.Seconds)
			{
				base.Dispatcher.Invoke(new Action(delegate
				{
					string text = this.FindResource("Button_Close") as string;
					this.close_btn.Content = string.Format("{0} ({1:00})", text, this._timeSpan.Seconds - interval);
				}), new object[0]);
				return;
			}
			base.Dispatcher.Invoke(new Action(delegate
			{
				this.Dispose();
			}), new object[0]);
		}

		// Token: 0x06000128 RID: 296 RVA: 0x00006374 File Offset: 0x00004574
		private void CloseButton_Click(object sender, RoutedEventArgs e)
		{
			this.Dispose();
		}

		// Token: 0x06000129 RID: 297 RVA: 0x0000637C File Offset: 0x0000457C
		public void Dispose()
		{
			this._timer.Stop();
			base.Close();
		}

		// Token: 0x04000077 RID: 119
		public static readonly DependencyProperty MessageProperty = DependencyProperty.Register("MessageProperty", typeof(string), typeof(AutoCloseDialog), new PropertyMetadata(string.Empty, new PropertyChangedCallback(AutoCloseDialog.OnMessageChanged)));

		// Token: 0x04000078 RID: 120
		private Timer _timer;

		// Token: 0x04000079 RID: 121
		private TimeSpan _timeSpan = new TimeSpan(0, 0, 10);

		// Token: 0x0400007A RID: 122
		private DateTime _startTime;
	}
}
