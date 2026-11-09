using System;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x02000003 RID: 3
	public static class TreeViewItemBehavior
	{
		// Token: 0x06000004 RID: 4 RVA: 0x000020B4 File Offset: 0x000002B4
		public static bool GetIsBroughtIntoViewWhenSelected(TreeViewItem treeViewItem)
		{
			return (bool)treeViewItem.GetValue(TreeViewItemBehavior.IsBroughtIntoViewWhenSelectedProperty);
		}

		// Token: 0x06000005 RID: 5 RVA: 0x000020C6 File Offset: 0x000002C6
		public static void SetIsBroughtIntoViewWhenSelected(TreeViewItem treeViewItem, bool value)
		{
			treeViewItem.SetValue(TreeViewItemBehavior.IsBroughtIntoViewWhenSelectedProperty, value);
		}

		// Token: 0x06000006 RID: 6 RVA: 0x000020DC File Offset: 0x000002DC
		private static void OnIsBroughtIntoViewWhenSelectedChanged(DependencyObject depObj, DependencyPropertyChangedEventArgs e)
		{
			TreeViewItem treeViewItem = depObj as TreeViewItem;
			if (treeViewItem == null)
			{
				return;
			}
			if (!(e.NewValue is bool))
			{
				return;
			}
			if ((bool)e.NewValue)
			{
				treeViewItem.Selected += TreeViewItemBehavior.OnTreeViewItemSelected;
				return;
			}
			treeViewItem.Selected -= TreeViewItemBehavior.OnTreeViewItemSelected;
		}

		// Token: 0x06000007 RID: 7 RVA: 0x00002138 File Offset: 0x00000338
		private static void OnTreeViewItemSelected(object sender, RoutedEventArgs e)
		{
			if (!object.ReferenceEquals(sender, e.OriginalSource))
			{
				return;
			}
			TreeViewItem treeViewItem = e.OriginalSource as TreeViewItem;
			if (treeViewItem != null)
			{
				treeViewItem.BringIntoView();
			}
		}

		// Token: 0x04000001 RID: 1
		public static readonly DependencyProperty IsBroughtIntoViewWhenSelectedProperty = DependencyProperty.RegisterAttached("IsBroughtIntoViewWhenSelected", typeof(bool), typeof(TreeViewItemBehavior), new UIPropertyMetadata(false, new PropertyChangedCallback(TreeViewItemBehavior.OnIsBroughtIntoViewWhenSelectedChanged)));
	}
}
