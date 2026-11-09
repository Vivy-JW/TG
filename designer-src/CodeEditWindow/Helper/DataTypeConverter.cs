using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000036 RID: 54
	public class DataTypeConverter : IValueConverter
	{
		// Token: 0x0600022E RID: 558 RVA: 0x00011195 File Offset: 0x0000F395
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value != null)
			{
				return value.GetType().Name;
			}
			return null;
		}

		// Token: 0x0600022F RID: 559 RVA: 0x000111A7 File Offset: 0x0000F3A7
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
