using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000047 RID: 71
	public class LineVisibilityConverter : IValueConverter
	{
		// Token: 0x060001C4 RID: 452 RVA: 0x0000C3BC File Offset: 0x0000A5BC
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

		// Token: 0x060001C5 RID: 453 RVA: 0x0000C466 File Offset: 0x0000A666
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
