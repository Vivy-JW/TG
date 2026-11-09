using System;
using System.Globalization;
using System.Windows;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200003C RID: 60
	public class NullVisibilityConveter : ConverterMarkupExtension<NullVisibilityConveter>
	{
		// Token: 0x0600018B RID: 395 RVA: 0x0000B1A6 File Offset: 0x000093A6
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return Visibility.Collapsed;
			}
			return Visibility.Visible;
		}

		// Token: 0x0600018C RID: 396 RVA: 0x0000B1B8 File Offset: 0x000093B8
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return Visibility.Visible;
		}
	}
}
