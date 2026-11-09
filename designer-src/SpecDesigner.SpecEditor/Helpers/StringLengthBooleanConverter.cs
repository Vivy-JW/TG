using System;
using System.Globalization;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000025 RID: 37
	public class StringLengthBooleanConverter : ConverterMarkupExtension<StringLengthBooleanConverter>
	{
		// Token: 0x06000110 RID: 272 RVA: 0x00009773 File Offset: 0x00007973
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (!(value is string))
			{
				return false;
			}
			return (value as string).Length > 0;
		}

		// Token: 0x06000111 RID: 273 RVA: 0x0000979B File Offset: 0x0000799B
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
