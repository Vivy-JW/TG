using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesignerCommon;

namespace SpecDesigner.Main
{
	// Token: 0x0200001E RID: 30
	public partial class SpecDesignerSplashScreen : Window
	{
		// Token: 0x0600022F RID: 559 RVA: 0x0000A850 File Offset: 0x00008A50
		public SpecDesignerSplashScreen(SpecDesignerInit specDesignerInit)
		{
			this.InitializeComponent();
			if (specDesignerInit == null)
			{
				return;
			}
			this.specDesignerInit = specDesignerInit;
			this.specDesignerInit.Completed += this.specDesignerInit_Completed;
			base.Loaded += this.SpecDesignerSplashScreen_Loaded;
		}

		// Token: 0x06000230 RID: 560 RVA: 0x0000A8A0 File Offset: 0x00008AA0
		private void specDesignerInit_Completed(object sender, EventArgs e)
		{
			if (base.DialogResult != null)
			{
				return;
			}
			base.DialogResult = new bool?(true);
		}

		// Token: 0x06000231 RID: 561 RVA: 0x0000A8CA File Offset: 0x00008ACA
		private void SpecDesignerSplashScreen_Loaded(object sender, RoutedEventArgs e)
		{
			SettingManager.Get().ProgressChanged += this.LoadCommonDataProgressChanged;
			this.specDesignerInit.Start(this);
		}

		// Token: 0x06000232 RID: 562 RVA: 0x0000A920 File Offset: 0x00008B20
		public void LoadCommonDataProgressChanged(int idx, int count, string file)
		{
			base.Dispatcher.BeginInvoke(new Action(delegate
			{
				this.proginfo.Text = "loading " + file + " .......";
			}), new object[0]);
		}

		// Token: 0x040000F0 RID: 240
		private SpecDesignerInit specDesignerInit;
	}
}
