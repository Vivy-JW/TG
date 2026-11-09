using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200002B RID: 43
	public class HelpCodeVisibleConverter : IValueConverter
	{
		// Token: 0x06000129 RID: 297 RVA: 0x00009BC1 File Offset: 0x00007DC1
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return Visibility.Collapsed;
		}

		// Token: 0x0600012A RID: 298 RVA: 0x00009BC9 File Offset: 0x00007DC9
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
