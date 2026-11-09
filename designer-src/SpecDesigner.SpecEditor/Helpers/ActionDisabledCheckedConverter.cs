using System;
using System.Globalization;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000019 RID: 25
	public class ActionDisabledCheckedConverter : ConverterMarkupExtension<ActionDisabledCheckedConverter>
	{
		// Token: 0x060000A9 RID: 169 RVA: 0x0000775C File Offset: 0x0000595C
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return false;
			}
			string text = (string)value;
			text = text.Trim();
			if (string.IsNullOrEmpty(text))
			{
				return false;
			}
			if (text.Equals(ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE)))
			{
				return true;
			}
			if (text.Equals(ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)))
			{
				return true;
			}
			return false;
		}

		// Token: 0x060000AA RID: 170 RVA: 0x000077CC File Offset: 0x000059CC
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return SpecStatus.MODIFY;
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
