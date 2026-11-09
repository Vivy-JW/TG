using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Windows.Data;
using System.Xml.Linq;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x0200001D RID: 29
	public class ColsConverter : IValueConverter
	{
		// Token: 0x060000D7 RID: 215 RVA: 0x00005CA4 File Offset: 0x00003EA4
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			FglParameter fglParameter = value as FglParameter;
			if (fglParameter == null || fglParameter.DB == null)
			{
				return null;
			}
			XElement xelement = TableColumnHelper.FindTableColumns(fglParameter.DB);
			if (xelement == null)
			{
				return null;
			}
			IEnumerable<string> enumerable = from ele in xelement.Elements("column")
				select ele.Attribute("name").Value;
			return enumerable.ToList<string>();
		}

		// Token: 0x060000D8 RID: 216 RVA: 0x00005D0F File Offset: 0x00003F0F
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
