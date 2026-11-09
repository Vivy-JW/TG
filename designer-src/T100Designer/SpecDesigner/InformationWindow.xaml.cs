using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Deployment.Application;
using System.Diagnostics;
using System.Reflection;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Markup;

namespace SpecDesigner
{
	// Token: 0x02000032 RID: 50
	public partial class InformationWindow : Window
	{
		// Token: 0x06000291 RID: 657 RVA: 0x0000BF50 File Offset: 0x0000A150
		public InformationWindow()
		{
			this.InitializeComponent();
			if (ApplicationDeployment.IsNetworkDeployed)
			{
				ApplicationDeployment currentDeployment = ApplicationDeployment.CurrentDeployment;
				string text = currentDeployment.CurrentVersion.ToString();
				this.verNum.Content = text;
				return;
			}
			this.verNum.Content = Assembly.GetExecutingAssembly().GetName().Version.ToString();
		}

		// Token: 0x06000292 RID: 658 RVA: 0x0000BFAE File Offset: 0x0000A1AE
		public new void Show()
		{
			base.Owner = Application.Current.MainWindow;
			base.ShowDialog();
		}

		// Token: 0x06000293 RID: 659 RVA: 0x0000BFC7 File Offset: 0x0000A1C7
		private void Button_Click(object sender, RoutedEventArgs e)
		{
		}
	}
}
