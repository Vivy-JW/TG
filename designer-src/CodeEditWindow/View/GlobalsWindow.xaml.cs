using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000014 RID: 20
	public partial class GlobalsWindow : Window
	{
		// Token: 0x0600008B RID: 139 RVA: 0x00005B1A File Offset: 0x00003D1A
		public GlobalsWindow()
		{
			this.InitializeComponent();
			base.DataContext = SettingManager.Get().Info_Globals;
		}

		// Token: 0x0600008C RID: 140 RVA: 0x00005B38 File Offset: 0x00003D38
		public new void Show()
		{
			base.Owner = Application.Current.MainWindow;
			base.ShowDialog();
		}

		// Token: 0x0600008D RID: 141 RVA: 0x00005B51 File Offset: 0x00003D51
		private void Close_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(true);
			base.Close();
		}
	}
}
