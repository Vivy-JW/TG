using System;
using System.Globalization;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200002C RID: 44
	public class CantDelTagReadonlyConverter : ConverterMarkupExtension<CantDelTagReadonlyConverter>
	{
		// Token: 0x0600012C RID: 300 RVA: 0x00009BD8 File Offset: 0x00007DD8
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value is bool)
			{
				bool flag = (bool)value;
				if (value != null)
				{
					return !(bool)value;
				}
			}
			return true;
		}

		// Token: 0x0600012D RID: 301 RVA: 0x00009C01 File Offset: 0x00007E01
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
