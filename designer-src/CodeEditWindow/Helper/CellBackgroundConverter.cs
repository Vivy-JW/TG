using System;
using System.Globalization;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Media;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000004 RID: 4
	public class CellBackgroundConverter : IValueConverter
	{
		// Token: 0x0600000A RID: 10 RVA: 0x00002114 File Offset: 0x00000314
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			DataGridCell dataGridCell = value as DataGridCell;
			if (dataGridCell == null || !dataGridCell.IsSelected)
			{
				return Brushes.Transparent;
			}
			if (dataGridCell.IsFocused)
			{
				return Brushes.Tomato;
			}
			return Brushes.SkyBlue;
		}

		// Token: 0x0600000B RID: 11 RVA: 0x0000214C File Offset: 0x0000034C
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
