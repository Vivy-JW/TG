using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon;

namespace SpecDesigner.Search.Helper
{
	// Token: 0x02000003 RID: 3
	public class MatchStringConverter : IValueConverter
	{
		// Token: 0x06000026 RID: 38 RVA: 0x00002A88 File Offset: 0x00000C88
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			SearchResultInfo searchResultInfo = value as SearchResultInfo;
			if (searchResultInfo != null)
			{
				if (searchResultInfo.Memo == -1)
				{
					return searchResultInfo.Match;
				}
				return string.Format("[{0}]{1}", searchResultInfo.Memo, searchResultInfo.Match);
			}
			else
			{
				ReplaceResultInfo replaceResultInfo = value as ReplaceResultInfo;
				if (replaceResultInfo == null)
				{
					return string.Empty;
				}
				if (replaceResultInfo.SearchInformation.Memo == -1)
				{
					return replaceResultInfo.SearchInformation.Match;
				}
				return string.Format("[{0}]{1}", replaceResultInfo.SearchInformation.Memo, replaceResultInfo.SearchInformation.Match);
			}
		}

		// Token: 0x06000027 RID: 39 RVA: 0x00002B1A File Offset: 0x00000D1A
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
