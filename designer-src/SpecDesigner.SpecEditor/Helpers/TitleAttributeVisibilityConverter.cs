using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000042 RID: 66
	internal class TitleAttributeVisibilityConverter : IMultiValueConverter
	{
		// Token: 0x060001B4 RID: 436 RVA: 0x0000C1F4 File Offset: 0x0000A3F4
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if ((string.Equals(values[2].ToString(), "Tree") || string.Equals(values[2].ToString(), "Table")) && !string.Equals("Phantom", values[1].ToString()))
			{
				return Visibility.Visible;
			}
			return Visibility.Collapsed;
		}

		// Token: 0x060001B5 RID: 437 RVA: 0x0000C24A File Offset: 0x0000A44A
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
