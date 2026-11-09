using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.Converters
{
	// Token: 0x02000005 RID: 5
	public class HistoryConverter : IValueConverter
	{
		// Token: 0x0600003E RID: 62 RVA: 0x00002D18 File Offset: 0x00000F18
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = value as string;
			if (!string.IsNullOrEmpty(text))
			{
				return text.Split(new char[] { ',' });
			}
			return string.Empty;
		}

		// Token: 0x0600003F RID: 63 RVA: 0x00002D4D File Offset: 0x00000F4D
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
