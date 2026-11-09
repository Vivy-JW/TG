using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000005 RID: 5
	public class ProgRelVisibleConverter : IValueConverter
	{
		// Token: 0x06000014 RID: 20 RVA: 0x00002302 File Offset: 0x00000502
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if ("Table".Equals(value) || "Tree".Equals(value))
			{
				return Visibility.Visible;
			}
			return Visibility.Collapsed;
		}

		// Token: 0x06000015 RID: 21 RVA: 0x0000232B File Offset: 0x0000052B
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
