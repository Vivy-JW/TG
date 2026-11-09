using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200004B RID: 75
	public class BooleanCheckedConverter : IValueConverter
	{
		// Token: 0x060001D0 RID: 464 RVA: 0x0000C7D4 File Offset: 0x0000A9D4
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return false;
			}
			bool? flag = null;
			string text = (string)value;
			text = text.Trim();
			if (text.Equals("true", StringComparison.CurrentCultureIgnoreCase))
			{
				flag = new bool?(true);
			}
			else
			{
				flag = new bool?(false);
			}
			return flag;
		}

		// Token: 0x060001D1 RID: 465 RVA: 0x0000C828 File Offset: 0x0000AA28
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
				text = "true";
			}
			else
			{
				text = "false";
			}
			return text;
		}
	}
}
