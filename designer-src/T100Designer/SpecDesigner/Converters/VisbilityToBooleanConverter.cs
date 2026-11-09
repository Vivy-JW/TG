using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.Converters
{
	// Token: 0x0200003A RID: 58
	public class VisbilityToBooleanConverter : IValueConverter
	{
		// Token: 0x060002C4 RID: 708 RVA: 0x0000CC14 File Offset: 0x0000AE14
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return (Visibility)value == Visibility.Visible;
		}

		// Token: 0x060002C5 RID: 709 RVA: 0x0000CC24 File Offset: 0x0000AE24
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return ((bool)value) ? Visibility.Visible : Visibility.Collapsed;
		}
	}
}
