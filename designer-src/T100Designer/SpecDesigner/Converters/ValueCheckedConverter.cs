using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.Converters
{
	// Token: 0x0200002D RID: 45
	public class ValueCheckedConverter : IValueConverter
	{
		// Token: 0x06000276 RID: 630 RVA: 0x0000BA14 File Offset: 0x00009C14
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return false;
			}
			bool? flag = null;
			string text = (string)value;
			text = text.Trim();
			if (text.Equals("Y", StringComparison.CurrentCultureIgnoreCase))
			{
				flag = new bool?(true);
			}
			else if (text.Equals("N", StringComparison.CurrentCultureIgnoreCase))
			{
				flag = new bool?(false);
			}
			return flag;
		}

		// Token: 0x06000277 RID: 631 RVA: 0x0000BA75 File Offset: 0x00009C75
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
