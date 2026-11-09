using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x0200003B RID: 59
	public class YesNoConverter : IValueConverter
	{
		// Token: 0x0600016D RID: 365 RVA: 0x00007058 File Offset: 0x00005258
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = value.ToString();
			if (text.Equals("Y", StringComparison.InvariantCultureIgnoreCase))
			{
				return true;
			}
			return false;
		}

		// Token: 0x0600016E RID: 366 RVA: 0x00007087 File Offset: 0x00005287
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
