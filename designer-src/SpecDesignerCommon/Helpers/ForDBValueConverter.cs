using System;
using System.Globalization;
using System.Linq;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x020000EC RID: 236
	public class ForDBValueConverter : IValueConverter
	{
		// Token: 0x060007EB RID: 2027 RVA: 0x000235E4 File Offset: 0x000217E4
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			int valueInt = -1;
			if (int.TryParse(value.ToString(), out valueInt))
			{
				DBOptions dboptions = SettingManager.Get().ForDBs.Where<DBOptions>((DBOptions db) => db.Value == valueInt).ElementAtOrDefault<DBOptions>(0);
				if (dboptions != null)
				{
					return dboptions.Description;
				}
			}
			return null;
		}

		// Token: 0x060007EC RID: 2028 RVA: 0x00023668 File Offset: 0x00021868
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value != null)
			{
				DBOptions dboptions = SettingManager.Get().ForDBs.Where<DBOptions>((DBOptions db) => db.Description.Equals(value.ToString())).ElementAtOrDefault<DBOptions>(0);
				if (dboptions != null)
				{
					return dboptions.Value;
				}
			}
			return -1;
		}
	}
}
