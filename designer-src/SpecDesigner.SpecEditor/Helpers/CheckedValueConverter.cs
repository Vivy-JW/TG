using System;
using System.Globalization;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000035 RID: 53
	public class CheckedValueConverter : ConverterMarkupExtension<CheckedValueConverter>
	{
		// Token: 0x06000159 RID: 345 RVA: 0x0000A598 File Offset: 0x00008798
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
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

		// Token: 0x0600015A RID: 346 RVA: 0x0000A608 File Offset: 0x00008808
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
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
