using System;
using System.Globalization;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200002A RID: 42
	internal class PreviewButtonEnabledMultiBindingConverter : IMultiValueConverter
	{
		// Token: 0x06000126 RID: 294 RVA: 0x00009B8C File Offset: 0x00007D8C
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if (values == null)
			{
				return false;
			}
			return (bool)values[0] && (bool)values[1];
		}

		// Token: 0x06000127 RID: 295 RVA: 0x00009BB2 File Offset: 0x00007DB2
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
