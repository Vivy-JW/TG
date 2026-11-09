using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon;

namespace SpecDesigner.Search.Helper
{
	// Token: 0x0200000A RID: 10
	public class MatchColorConverter : IValueConverter
	{
		// Token: 0x0600008D RID: 141 RVA: 0x00003F74 File Offset: 0x00002174
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			SearchResultInfo searchResultInfo = value as SearchResultInfo;
			if (searchResultInfo.ComparingSame)
			{
				return true;
			}
			return false;
		}

		// Token: 0x0600008E RID: 142 RVA: 0x00003F9D File Offset: 0x0000219D
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
