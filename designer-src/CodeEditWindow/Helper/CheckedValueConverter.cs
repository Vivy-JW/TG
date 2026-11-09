using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesigner.Infrastructure;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000032 RID: 50
	public class CheckedValueConverter : IValueConverter
	{
		// Token: 0x060001ED RID: 493 RVA: 0x0000F988 File Offset: 0x0000DB88
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
			return ResourceController.GetInstance().GetProgramInfo(packageKey).IsTopstdMode;
		}

		// Token: 0x060001EE RID: 494 RVA: 0x0000F9DB File Offset: 0x0000DBDB
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return null;
		}
	}
}
