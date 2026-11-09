using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000045 RID: 69
	public class TreeViewItemToLineTranslateTransformFactor : IValueConverter
	{
		// Token: 0x060001BE RID: 446 RVA: 0x0000C328 File Offset: 0x0000A528
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return Math.Ceiling((double)value / 2.0) + 3.0;
		}

		// Token: 0x060001BF RID: 447 RVA: 0x0000C34E File Offset: 0x0000A54E
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
