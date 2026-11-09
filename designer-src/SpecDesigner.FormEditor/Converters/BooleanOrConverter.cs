using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x0200001A RID: 26
	public class BooleanOrConverter : IMultiValueConverter
	{
		// Token: 0x060000D6 RID: 214 RVA: 0x0000531C File Offset: 0x0000351C
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			foreach (object obj in values)
			{
				if (obj is bool && (bool)obj)
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x060000D7 RID: 215 RVA: 0x0000535E File Offset: 0x0000355E
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotSupportedException("BooleanAndConverter is a OneWay converter.");
		}
	}
}
