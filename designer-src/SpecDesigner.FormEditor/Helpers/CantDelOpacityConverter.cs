using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000047 RID: 71
	public class CantDelOpacityConverter : IValueConverter
	{
		// Token: 0x06000298 RID: 664 RVA: 0x0000DB51 File Offset: 0x0000BD51
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return 1;
			}
			if ((bool)value)
			{
				return 0.5;
			}
			return 1;
		}

		// Token: 0x06000299 RID: 665 RVA: 0x0000DB7A File Offset: 0x0000BD7A
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
