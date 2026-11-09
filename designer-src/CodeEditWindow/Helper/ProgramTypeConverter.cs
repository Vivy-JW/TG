using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000008 RID: 8
	public class ProgramTypeConverter : IValueConverter
	{
		// Token: 0x06000047 RID: 71 RVA: 0x00004B0F File Offset: 0x00002D0F
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return false;
			}
			if (value.Equals("M") || value.Equals("S") || value.Equals("Z"))
			{
				return true;
			}
			return false;
		}

		// Token: 0x06000048 RID: 72 RVA: 0x00004B4F File Offset: 0x00002D4F
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
