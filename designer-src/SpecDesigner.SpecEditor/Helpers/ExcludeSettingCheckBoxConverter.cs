using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000020 RID: 32
	public class ExcludeSettingCheckBoxConverter : IValueConverter
	{
		// Token: 0x060000FB RID: 251 RVA: 0x00009442 File Offset: 0x00007642
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return false;
		}

		// Token: 0x060000FC RID: 252 RVA: 0x0000944A File Offset: 0x0000764A
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if ((bool)value)
			{
				return SpecStatus.MODIFY;
			}
			return SpecStatus.DELETE;
		}
	}
}
