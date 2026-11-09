using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesigner.FormEditor.Test
{
	// Token: 0x02000003 RID: 3
	public partial class TestPropertyWindow : Window
	{
		// Token: 0x06000006 RID: 6 RVA: 0x0000217D File Offset: 0x0000037D
		public TestPropertyWindow()
		{
			this.InitializeComponent();
			Application.Current.MainWindow.Closed += this.MainWindow_Closed;
		}

		// Token: 0x06000007 RID: 7 RVA: 0x000021A6 File Offset: 0x000003A6
		private void MainWindow_Closed(object sender, EventArgs e)
		{
			TestPropertyWindow.win.Closing -= TestPropertyWindow.win_Closing;
			TestPropertyWindow.win.Close();
		}

		// Token: 0x06000008 RID: 8 RVA: 0x000021C8 File Offset: 0x000003C8
		public static void Show(string form, string tsd)
		{
			if (TestPropertyWindow.win == null)
			{
				TestPropertyWindow.win = new TestPropertyWindow();
				TestPropertyWindow.win.Closing += TestPropertyWindow.win_Closing;
			}
			TestPropertyWindow.win.formScrollViewer.DataContext = form;
			TestPropertyWindow.win.tsdScrollViewer.DataContext = tsd;
			TestPropertyWindow.win.Show();
		}

		// Token: 0x06000009 RID: 9 RVA: 0x00002226 File Offset: 0x00000426
		private static void win_Closing(object sender, CancelEventArgs e)
		{
			e.Cancel = true;
			(sender as Window).Hide();
		}

		// Token: 0x0600000A RID: 10 RVA: 0x0000223A File Offset: 0x0000043A
		private void Copy4FDBtn_Click(object sender, RoutedEventArgs e)
		{
			Clipboard.SetDataObject(TestPropertyWindow.win.formScrollViewer.DataContext as string);
		}

		// Token: 0x0600000B RID: 11 RVA: 0x00002255 File Offset: 0x00000455
		private void Copy4TSDtn_Click(object sender, RoutedEventArgs e)
		{
			Clipboard.SetDataObject(TestPropertyWindow.win.tsdScrollViewer.DataContext as string);
		}

		// Token: 0x0400000B RID: 11
		private static TestPropertyWindow win;
	}
}
