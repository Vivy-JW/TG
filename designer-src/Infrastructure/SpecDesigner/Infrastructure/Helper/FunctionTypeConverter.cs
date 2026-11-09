using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x02000002 RID: 2
	public class FunctionTypeConverter : IValueConverter
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = value.ToString();
			if (text.Equals("PUBLIC", StringComparison.CurrentCultureIgnoreCase))
			{
				return 0;
			}
			return 1;
		}

		// Token: 0x06000002 RID: 2 RVA: 0x00002080 File Offset: 0x00000280
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			int num = 0;
			if (int.TryParse(value.ToString(), out num) && num == 0)
			{
				return "PUBLIC";
			}
			return "PRIVATE";
		}
	}
}
