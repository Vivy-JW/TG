using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x0200004B RID: 75
	public class CheckedValueConverter : IValueConverter
	{
		// Token: 0x060002C5 RID: 709 RVA: 0x0000E0B0 File Offset: 0x0000C2B0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return false;
			}
			if (!(value is string))
			{
				return false;
			}
			bool? flag = null;
			string text = (string)value;
			text = text.Trim();
			if (text.Equals("Y", StringComparison.CurrentCultureIgnoreCase) || text.Equals("TRUE", StringComparison.CurrentCultureIgnoreCase))
			{
				flag = new bool?(true);
			}
			else
			{
				flag = new bool?(false);
			}
			return flag;
		}

		// Token: 0x060002C6 RID: 710 RVA: 0x0000E120 File Offset: 0x0000C320
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
				text = "Y";
			}
			else
			{
				text = "N";
			}
			return text;
		}
	}
}
