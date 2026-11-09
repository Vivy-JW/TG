using System;
using System.Globalization;
using System.Linq;
using System.Windows.Data;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000018 RID: 24
	public class RequiredStyleToBooleanConverter : IValueConverter
	{
		// Token: 0x060000CF RID: 207 RVA: 0x00005233 File Offset: 0x00003433
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return RequiredStyleToBooleanConverter.CovertToBool(value);
		}

		// Token: 0x060000D0 RID: 208 RVA: 0x0000523B File Offset: 0x0000343B
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}

		// Token: 0x060000D1 RID: 209 RVA: 0x00005244 File Offset: 0x00003444
		public static object CovertToBool(object value)
		{
			string text = value as string;
			if (string.IsNullOrEmpty(text))
			{
				return false;
			}
			if (text.Trim().Length == 0)
			{
				return false;
			}
			if (text.Split(new char[] { ' ' }).ToList<string>().Contains("required"))
			{
				return true;
			}
			return false;
		}
	}
}
