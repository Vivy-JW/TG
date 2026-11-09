using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.TableViewer.Helper
{
	// Token: 0x02000005 RID: 5
	public class BooleanConverter : IValueConverter
	{
		// Token: 0x06000021 RID: 33 RVA: 0x00002D80 File Offset: 0x00000F80
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = value.ToString();
			if (string.Equals(text, "y", StringComparison.InvariantCultureIgnoreCase))
			{
				return true;
			}
			return false;
		}

		// Token: 0x06000022 RID: 34 RVA: 0x00002DAF File Offset: 0x00000FAF
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
