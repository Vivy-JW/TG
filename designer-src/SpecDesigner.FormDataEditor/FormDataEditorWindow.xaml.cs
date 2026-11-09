using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesigner.FormDataEditor
{
	// Token: 0x02000007 RID: 7
	public partial class FormDataEditorWindow : Window
	{
		// Token: 0x06000013 RID: 19 RVA: 0x000024B6 File Offset: 0x000006B6
		public FormDataEditorWindow()
		{
			this.InitializeComponent();
			base.Title = Application.Current.FindResource("dataView_WindowTitle") as string;
		}

		// Token: 0x06000014 RID: 20 RVA: 0x000024DE File Offset: 0x000006DE
		public new void Show()
		{
			base.Owner = Application.Current.MainWindow;
			base.ShowDialog();
		}

		// Token: 0x06000015 RID: 21 RVA: 0x000024F7 File Offset: 0x000006F7
		private void Button_Click(object sender, RoutedEventArgs e)
		{
		}
	}
}
