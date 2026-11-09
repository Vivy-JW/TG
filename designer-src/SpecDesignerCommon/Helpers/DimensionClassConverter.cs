using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Globalization;
using System.Linq;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x0200003A RID: 58
	public class DimensionClassConverter : IValueConverter
	{
		// Token: 0x060001E3 RID: 483 RVA: 0x00008BF0 File Offset: 0x00006DF0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			FglDimension fglDimension = value as FglDimension;
			IEnumerable<DimensionOption> enumerable = SettingManager.Get().Dimensions.Where<DimensionOption>((DimensionOption d) => d.No == fglDimension.No);
			ObservableCollection<DimensionClassOption> collection = new ObservableCollection<DimensionClassOption>();
			foreach (DimensionOption dimensionOption in enumerable)
			{
				dimensionOption.Class.ForEach(delegate(DimensionClassOption c)
				{
					collection.Add(c);
				});
			}
			return collection;
		}

		// Token: 0x060001E4 RID: 484 RVA: 0x00008C94 File Offset: 0x00006E94
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
