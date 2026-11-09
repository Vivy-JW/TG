using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000046 RID: 70
	public class FirstLeafLineMarginConverter : IValueConverter
	{
		// Token: 0x060001C1 RID: 449 RVA: 0x0000C360 File Offset: 0x0000A560
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return new Thickness(0.0, 1.0, 0.0, (double)value / 2.0 - 3.0);
		}

		// Token: 0x060001C2 RID: 450 RVA: 0x0000C3AC File Offset: 0x0000A5AC
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
