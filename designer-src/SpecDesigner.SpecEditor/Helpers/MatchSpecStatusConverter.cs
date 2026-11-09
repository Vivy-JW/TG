using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000044 RID: 68
	public class MatchSpecStatusConverter : IValueConverter
	{
		// Token: 0x060001BB RID: 443 RVA: 0x0000C2DC File Offset: 0x0000A4DC
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			SpecStatus specStatus = (SpecStatus)value;
			if (value == null)
			{
				return false;
			}
			SpecStatus specStatus2 = (SpecStatus)parameter;
			if (value == null)
			{
				return false;
			}
			return (specStatus & specStatus2) != SpecStatus.NULL;
		}

		// Token: 0x060001BC RID: 444 RVA: 0x0000C319 File Offset: 0x0000A519
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
