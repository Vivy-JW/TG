using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200003A RID: 58
	internal class TableAssociationConverter : IMultiValueConverter
	{
		// Token: 0x06000185 RID: 389 RVA: 0x0000B110 File Offset: 0x00009310
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if ("Visibility".Equals(parameter))
			{
				string text = values[0].ToString();
				if ("Tree".Equals(text) || "Table".Equals(text))
				{
					return Visibility.Visible;
				}
				return Visibility.Collapsed;
			}
			else
			{
				if ("ItemsSource".Equals(parameter) && values != null && !string.IsNullOrEmpty(values[0].ToString()))
				{
					object obj = values[0];
					return null;
				}
				return null;
			}
		}

		// Token: 0x06000186 RID: 390 RVA: 0x0000B181 File Offset: 0x00009381
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
