using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x02000036 RID: 54
	public class TabIndexVisibilityConverter : IMultiValueConverter
	{
		// Token: 0x06000200 RID: 512 RVA: 0x0000AE2C File Offset: 0x0000902C
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			bool flag = false;
			bool.TryParse(values[0].ToString(), out flag);
			if (!flag)
			{
				return false;
			}
			bool flag2 = false;
			bool.TryParse(values[1].ToString(), out flag2);
			if (!flag2)
			{
				return false;
			}
			return true;
		}

		// Token: 0x06000201 RID: 513 RVA: 0x0000AE77 File Offset: 0x00009077
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
