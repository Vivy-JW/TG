using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000018 RID: 24
	public class CheckedStatusConverter : IValueConverter
	{
		// Token: 0x060000A6 RID: 166 RVA: 0x000076C0 File Offset: 0x000058C0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return false;
			}
			bool? flag = null;
			string text = (string)value;
			text = text.Trim();
			if (text.Equals(ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE), StringComparison.CurrentCultureIgnoreCase))
			{
				flag = new bool?(true);
			}
			else
			{
				flag = new bool?(false);
			}
			return flag;
		}

		// Token: 0x060000A7 RID: 167 RVA: 0x0000771C File Offset: 0x0000591C
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return null;
			}
			bool flag = (bool)value;
			string text;
			if (flag)
			{
				text = ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE);
			}
			else
			{
				text = ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY);
			}
			return text;
		}
	}
}
