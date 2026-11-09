using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesignerCommon.Views
{
	// Token: 0x0200005D RID: 93
	public partial class LocalItemsSelectionWindow : Window
	{
		// Token: 0x06000320 RID: 800 RVA: 0x0000CDD2 File Offset: 0x0000AFD2
		public LocalItemsSelectionWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000321 RID: 801 RVA: 0x0000CDE0 File Offset: 0x0000AFE0
		private void OKButton_Click(object sender, RoutedEventArgs e)
		{
			base.Close();
		}
	}
}
