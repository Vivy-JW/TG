using System;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x0200003E RID: 62
	public class TreeViewItemHelper
	{
		// Token: 0x06000229 RID: 553 RVA: 0x0000BD28 File Offset: 0x00009F28
		public static void OnSelectedAndScrolling(DependencyObject sender, DependencyPropertyChangedEventArgs e)
		{
			TreeViewItem treeViewItem = sender as TreeViewItem;
			if (treeViewItem == null)
			{
				return;
			}
			bool flag = false;
			bool.TryParse(e.NewValue.ToString(), out flag);
			if (flag)
			{
				treeViewItem.BringIntoView();
			}
		}

		// Token: 0x0600022A RID: 554 RVA: 0x0000BD5F File Offset: 0x00009F5F
		public static void SetIsSelectedAndScrolling(DependencyObject obj, bool value)
		{
			obj.SetValue(TreeViewItemHelper.IsSelectedAndScrollingProperty, value);
		}

		// Token: 0x0600022B RID: 555 RVA: 0x0000BD72 File Offset: 0x00009F72
		public static bool GetIsSelectedAndScrolling(DependencyObject obj)
		{
			return (bool)obj.GetValue(TreeViewItemHelper.IsSelectedAndScrollingProperty);
		}

		// Token: 0x04000131 RID: 305
		public static readonly DependencyProperty IsSelectedAndScrollingProperty = DependencyProperty.RegisterAttached("IsSelectedAndScrolling", typeof(bool), typeof(TreeViewItemHelper), new PropertyMetadata(false, new PropertyChangedCallback(TreeViewItemHelper.OnSelectedAndScrolling)));
	}
}
