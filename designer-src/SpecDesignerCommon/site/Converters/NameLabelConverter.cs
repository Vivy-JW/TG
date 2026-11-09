using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesignerCommon.Site.Converters
{
	// Token: 0x02000116 RID: 278
	public class NameLabelConverter : IMultiValueConverter
	{
		// Token: 0x060009E9 RID: 2537 RVA: 0x000316E0 File Offset: 0x0002F8E0
		public object Convert(object[] value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return null;
			}
			if (value.Length != 2)
			{
				return null;
			}
			string text = "{0} (ver:{1})";
			if (value[1] == DependencyProperty.UnsetValue)
			{
				text = "{0}";
			}
			return string.Format(text, value[0], value[1]);
		}

		// Token: 0x060009EA RID: 2538 RVA: 0x0003171C File Offset: 0x0002F91C
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
