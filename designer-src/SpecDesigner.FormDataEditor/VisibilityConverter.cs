using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;

namespace SpecDesigner.FormDataEditor
{
	// Token: 0x02000003 RID: 3
	public class VisibilityConverter : IValueConverter
	{
		// Token: 0x06000004 RID: 4 RVA: 0x00002078 File Offset: 0x00000278
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return Visibility.Collapsed;
			}
			if (value is DetailView && !(value as DetailView).HasItems)
			{
				return Visibility.Collapsed;
			}
			if (value is DataGrid && !(value as DataGrid).HasItems)
			{
				return Visibility.Collapsed;
			}
			if (string.IsNullOrEmpty(value.ToString()))
			{
				return Visibility.Collapsed;
			}
			return Visibility.Visible;
		}

		// Token: 0x06000005 RID: 5 RVA: 0x000020E1 File Offset: 0x000002E1
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
