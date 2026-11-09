using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000019 RID: 25
	public class LocalStringsConverter : IMultiValueConverter
	{
		// Token: 0x060000D3 RID: 211 RVA: 0x000052B4 File Offset: 0x000034B4
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if (values[0] == DependencyProperty.UnsetValue || values[1] == DependencyProperty.UnsetValue)
			{
				return null;
			}
			string text = values[0].ToString();
			values[1].ToString();
			string columnTextByFullName = TableColumnHelper.GetColumnTextByFullName(text);
			if (columnTextByFullName == "")
			{
				return text;
			}
			return string.Format("{0}  ({1})", text, columnTextByFullName);
		}

		// Token: 0x060000D4 RID: 212 RVA: 0x0000530B File Offset: 0x0000350B
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
