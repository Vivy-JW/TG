using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x0200004C RID: 76
	public class FormVisibilityConverter : IMultiValueConverter
	{
		// Token: 0x060002C8 RID: 712 RVA: 0x0000E154 File Offset: 0x0000C354
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			Visibility visibility = Visibility.Visible;
			if (!(bool)values[1] && ((string)values[0] == "worksheet" || (string)values[0] == "vb_quantity"))
			{
				visibility = Visibility.Collapsed;
			}
			return visibility;
		}

		// Token: 0x060002C9 RID: 713 RVA: 0x0000E19D File Offset: 0x0000C39D
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
