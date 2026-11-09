using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000017 RID: 23
	public class ResetBoldConverter : ConverterMarkupExtension<ResetBoldConverter>
	{
		// Token: 0x060000A3 RID: 163 RVA: 0x0000764C File Offset: 0x0000584C
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == Binding.DoNothing)
			{
				return FontWeights.Normal;
			}
			if (value == null)
			{
				return FontWeights.Normal;
			}
			if (value.ToString().Equals("[[RE][SET]]") || object.Equals(string.Empty, value))
			{
				return FontWeights.Normal;
			}
			return FontWeights.Bold;
		}

		// Token: 0x060000A4 RID: 164 RVA: 0x000076AE File Offset: 0x000058AE
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
