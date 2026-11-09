using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x02000005 RID: 5
	public class IsEqualOrGreaterThanConverter : IValueConverter
	{
		// Token: 0x06000017 RID: 23 RVA: 0x00002628 File Offset: 0x00000828
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			int num = (int)value;
			int num2 = int.Parse(parameter.ToString());
			return num >= num2;
		}

		// Token: 0x06000018 RID: 24 RVA: 0x00002654 File Offset: 0x00000854
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}

		// Token: 0x04000011 RID: 17
		public static readonly IValueConverter This = new IsEqualOrGreaterThanConverter();
	}
}
