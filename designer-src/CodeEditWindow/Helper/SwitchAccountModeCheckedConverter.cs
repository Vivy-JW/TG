using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesigner.Infrastructure;
using SpecDesignerCommon;
using SpecDesignerPreference;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200001C RID: 28
	internal class SwitchAccountModeCheckedConverter : IValueConverter
	{
		// Token: 0x0600010B RID: 267 RVA: 0x0000AE04 File Offset: 0x00009004
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			PackageKey packageKey = value as PackageKey;
			if (null == packageKey)
			{
				return false;
			}
			if (ResourceController.GetInstance().GetProgramInfo(packageKey) == null)
			{
				return false;
			}
			if (ResourceController.GetInstance().GetProgramInfo(packageKey).AllowChangeToTopstd && PreferenceManager.Current.Settings.TopstdEditPermission)
			{
				return true;
			}
			return false;
		}

		// Token: 0x0600010C RID: 268 RVA: 0x0000AE6C File Offset: 0x0000906C
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return null;
		}
	}
}
