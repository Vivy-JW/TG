using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesignerCommon.Site.Converters
{
	// Token: 0x0200004E RID: 78
	public class FirstLeafLineMarginConverter : IValueConverter
	{
		// Token: 0x0600029E RID: 670 RVA: 0x0000B994 File Offset: 0x00009B94
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return new Thickness(0.0, 1.0, 0.0, (double)value / 2.0 - 3.0);
		}

		// Token: 0x0600029F RID: 671 RVA: 0x0000B9E0 File Offset: 0x00009BE0
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
