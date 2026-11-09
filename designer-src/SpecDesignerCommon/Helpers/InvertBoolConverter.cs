using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000079 RID: 121
	[ValueConversion(typeof(bool), typeof(bool))]
	public class InvertBoolConverter : IValueConverter
	{
		// Token: 0x0600049A RID: 1178 RVA: 0x00014CE4 File Offset: 0x00012EE4
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			bool flag = (bool)value;
			return !flag;
		}

		// Token: 0x0600049B RID: 1179 RVA: 0x00014D04 File Offset: 0x00012F04
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			bool flag = (bool)value;
			return !flag;
		}
	}
}
