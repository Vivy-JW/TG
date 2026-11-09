using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesigner.Infrastructure;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200000F RID: 15
	public class SwitchAccountModeVisiblityConverter : IValueConverter
	{
		// Token: 0x06000079 RID: 121 RVA: 0x000057E4 File Offset: 0x000039E4
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			PackageKey packageKey = value as PackageKey;
			if (null == packageKey)
			{
				return Visibility.Collapsed;
			}
			if (ResourceController.GetInstance().GetProgramInfo(packageKey) == null)
			{
				return Visibility.Collapsed;
			}
			return ResourceController.GetInstance().GetProgramInfo(packageKey).AllowChangeToTopstd ? Visibility.Visible : Visibility.Collapsed;
		}

		// Token: 0x0600007A RID: 122 RVA: 0x00005837 File Offset: 0x00003A37
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return null;
		}
	}
}
