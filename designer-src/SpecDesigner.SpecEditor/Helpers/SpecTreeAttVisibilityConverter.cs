using System;
using System.Globalization;
using System.Linq;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000029 RID: 41
	public class SpecTreeAttVisibilityConverter : IMultiValueConverter
	{
		// Token: 0x06000123 RID: 291 RVA: 0x00009B0C File Offset: 0x00007D0C
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if (values[0] == DependencyProperty.UnsetValue || values[1] == DependencyProperty.UnsetValue)
			{
				return Visibility.Collapsed;
			}
			string text = values[1] as string;
			string text2 = values[0] as string;
			if (text2 == null)
			{
				return Visibility.Collapsed;
			}
			if (text2.Split(new char[] { ',' }).ToList<string>().Contains(text))
			{
				return Visibility.Visible;
			}
			return Visibility.Collapsed;
		}

		// Token: 0x06000124 RID: 292 RVA: 0x00009B7D File Offset: 0x00007D7D
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
