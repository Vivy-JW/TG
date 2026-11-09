using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x0200002B RID: 43
	public class DateTimeEditVisiblityConverter : IValueConverter
	{
		// Token: 0x0600018A RID: 394 RVA: 0x0000875E File Offset: 0x0000695E
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return (SettingManager.Get().ErpVer == "3.0") ? Visibility.Visible : Visibility.Collapsed;
		}

		// Token: 0x0600018B RID: 395 RVA: 0x0000877F File Offset: 0x0000697F
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return null;
		}
	}
}
