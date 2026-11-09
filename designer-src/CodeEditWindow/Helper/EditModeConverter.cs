using System;
using System.Globalization;
using System.Windows.Data;
using ICSharpCode.AvalonEdit;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000046 RID: 70
	public class EditModeConverter : IValueConverter
	{
		// Token: 0x060002F7 RID: 759 RVA: 0x00019ADC File Offset: 0x00017CDC
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			ContentType contentType = (ContentType)value;
			if (contentType == ContentType.ADP)
			{
				return false;
			}
			return true;
		}

		// Token: 0x060002F8 RID: 760 RVA: 0x00019B01 File Offset: 0x00017D01
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value.ToString() == bool.TrueString)
			{
				return ContentType.SEC;
			}
			return ContentType.ADP;
		}
	}
}
