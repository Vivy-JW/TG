using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesignerCommon.Site.Converters
{
	// Token: 0x0200004D RID: 77
	public class TreeViewItemToLineTranslateTransformFactor : IValueConverter
	{
		// Token: 0x0600029B RID: 667 RVA: 0x0000B95D File Offset: 0x00009B5D
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return Math.Ceiling((double)value / 2.0) + 3.0;
		}

		// Token: 0x0600029C RID: 668 RVA: 0x0000B983 File Offset: 0x00009B83
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
