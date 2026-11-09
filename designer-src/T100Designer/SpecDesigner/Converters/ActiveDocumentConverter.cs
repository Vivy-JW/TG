using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesigner.ViewModels;

namespace SpecDesigner.Converters
{
	// Token: 0x0200001F RID: 31
	internal class ActiveDocumentConverter : IValueConverter
	{
		// Token: 0x06000235 RID: 565 RVA: 0x0000A9B7 File Offset: 0x00008BB7
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value is FileViewModel)
			{
				return value;
			}
			return Binding.DoNothing;
		}

		// Token: 0x06000236 RID: 566 RVA: 0x0000A9C8 File Offset: 0x00008BC8
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value is FileViewModel)
			{
				return value;
			}
			return Binding.DoNothing;
		}
	}
}
