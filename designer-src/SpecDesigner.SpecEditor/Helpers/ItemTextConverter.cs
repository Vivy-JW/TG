using System;
using System.Globalization;
using System.Linq;
using System.Windows.Data;
using SpecDesignerCommon;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000048 RID: 72
	public class ItemTextConverter : IMultiValueConverter
	{
		// Token: 0x060001C7 RID: 455 RVA: 0x0000C478 File Offset: 0x0000A678
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if (values == null)
			{
				return null;
			}
			if (values.Count<object>() != 2)
			{
				return null;
			}
			PackageKey packageKey = values[1] as PackageKey;
			if (null == packageKey)
			{
				throw new Exception("Program Name can't be null");
			}
			return SettingManager.Get().GetTzpManger(packageKey).SpecificationInfo.GetFieldLocalStringText((values[1] as PackageKey).Program);
		}

		// Token: 0x060001C8 RID: 456 RVA: 0x0000C4D6 File Offset: 0x0000A6D6
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
