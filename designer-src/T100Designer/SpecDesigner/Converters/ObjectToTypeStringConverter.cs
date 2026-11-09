using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.Converters
{
	// Token: 0x02000039 RID: 57
	public class ObjectToTypeStringConverter : IValueConverter
	{
		// Token: 0x060002C1 RID: 705 RVA: 0x0000CBEF File Offset: 0x0000ADEF
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return "";
			}
			return value.GetType().ToString();
		}

		// Token: 0x060002C2 RID: 706 RVA: 0x0000CC05 File Offset: 0x0000AE05
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
