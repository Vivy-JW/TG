using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesignerPreference.Helper
{
	// Token: 0x0200000B RID: 11
	public class ContentConverter : IValueConverter
	{
		// Token: 0x0600005B RID: 91 RVA: 0x0000382B File Offset: 0x00001A2B
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return string.Empty;
			}
			return "r.r " + value;
		}

		// Token: 0x0600005C RID: 92 RVA: 0x00003841 File Offset: 0x00001A41
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
