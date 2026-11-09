using System;
using System.Collections.Generic;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000010 RID: 16
	public class MultiNullVisibilityConveter : IValueConverter
	{
		// Token: 0x0600006C RID: 108 RVA: 0x000064A0 File Offset: 0x000046A0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			List<XmlElement> list = value as List<XmlElement>;
			string text = parameter as string;
			if (text == null || list == null)
			{
				return Visibility.Collapsed;
			}
			foreach (XmlElement xmlElement in list)
			{
				if (xmlElement.GetAttribute(text) == null)
				{
					return Visibility.Collapsed;
				}
			}
			return Visibility.Visible;
		}

		// Token: 0x0600006D RID: 109 RVA: 0x00006520 File Offset: 0x00004720
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
