using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;

namespace SpecDesignerCommon.Site.Converters
{
	// Token: 0x0200004F RID: 79
	public class LineVisibilityConverter : IValueConverter
	{
		// Token: 0x060002A1 RID: 673 RVA: 0x0000B9F0 File Offset: 0x00009BF0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			Visibility visibility = Visibility.Collapsed;
			TreeViewItem treeViewItem = value as TreeViewItem;
			if (treeViewItem == null)
			{
				throw new ArgumentException("value");
			}
			if ((string)parameter == "Line")
			{
				Panel panel = treeViewItem.SafeFindAncestor<Panel>(null);
				if (panel != null)
				{
					int num = panel.Children.IndexOf(treeViewItem);
					if (num < panel.Children.Count - 1)
					{
						visibility = Visibility.Visible;
					}
				}
			}
			else if ((string)parameter == "FirstLeafLine")
			{
				Panel panel2 = treeViewItem.SafeFindAncestor<Panel>(null);
				if (panel2 != null && panel2.Children.IndexOf(treeViewItem) == 0)
				{
					TreeViewItem treeViewItem2 = panel2.SafeFindAncestor<TreeViewItem>(null);
					if (treeViewItem2 != null)
					{
						visibility = Visibility.Visible;
					}
				}
			}
			return visibility;
		}

		// Token: 0x060002A2 RID: 674 RVA: 0x0000BA9A File Offset: 0x00009C9A
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
