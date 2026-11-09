using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x0200001F RID: 31
	public class ObjectToTypeStringConverter : IValueConverter
	{
		// Token: 0x060000F8 RID: 248 RVA: 0x0000941C File Offset: 0x0000761C
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return null;
			}
			return value.GetType().Name;
		}

		// Token: 0x060000F9 RID: 249 RVA: 0x0000942E File Offset: 0x0000762E
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new Exception("Can't convert back");
		}
	}
}
