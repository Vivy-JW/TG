using System;
using System.Collections.Generic;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000038 RID: 56
	public class RadioGroupItemConverter : IMultiValueConverter
	{
		// Token: 0x06000206 RID: 518 RVA: 0x0000AF10 File Offset: 0x00009110
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if (values == null)
			{
				return null;
			}
			if (values.Length != 2)
			{
				return null;
			}
			IList<XmlElement> list = values[0] as IList<XmlElement>;
			string text = values[1] as string;
			if (list == null || text == null)
			{
				return null;
			}
			StackPanel stackPanel = new StackPanel();
			stackPanel.Orientation = ((text == "horizontal") ? Orientation.Horizontal : Orientation.Vertical);
			foreach (XmlElement xmlElement in list)
			{
				stackPanel.Children.Add(this.convertToRadioButton(xmlElement));
			}
			return stackPanel;
		}

		// Token: 0x06000207 RID: 519 RVA: 0x0000AFB0 File Offset: 0x000091B0
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			return null;
		}

		// Token: 0x06000208 RID: 520 RVA: 0x0000AFB4 File Offset: 0x000091B4
		private RadioButton convertToRadioButton(XmlElement element)
		{
			return new RadioButton
			{
				Content = element.GetAttribute("text"),
				IsHitTestVisible = false,
				Margin = new Thickness(2.0)
			};
		}
	}
}
