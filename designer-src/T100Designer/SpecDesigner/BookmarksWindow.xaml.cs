using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.ViewModels;
using SpecDesignerCommon.Bookmark;

namespace SpecDesigner
{
	// Token: 0x0200000B RID: 11
	public partial class BookmarksWindow : UserControl
	{
		// Token: 0x0600005C RID: 92 RVA: 0x00003097 File Offset: 0x00001297
		public BookmarksWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x0600005D RID: 93 RVA: 0x000030A8 File Offset: 0x000012A8
		private void dataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			FileViewModel fileViewModel = base.DataContext as FileViewModel;
			if (fileViewModel != null && fileViewModel.BookmarkManager != null)
			{
				fileViewModel.BookmarkManager.SelectedBookmark = null;
				fileViewModel.BookmarkManager.SelectedBookmark = this.dataGrid.CurrentItem as IBookmark;
			}
		}
	}
}
