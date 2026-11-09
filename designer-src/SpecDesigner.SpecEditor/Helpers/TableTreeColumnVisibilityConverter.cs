using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200000B RID: 11
	public class TableTreeColumnVisibilityConverter : IValueConverter
	{
		// Token: 0x0600005F RID: 95 RVA: 0x00005B3C File Offset: 0x00003D3C
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value is XmlElement)
			{
				XmlElement xmlElement = value as XmlElement;
				if (xmlElement.Type == ComponentType.Phantom)
				{
					return Visibility.Collapsed;
				}
				if (xmlElement.Parent != null && (xmlElement.Parent.Type == ComponentType.Table || xmlElement.Parent.Type == ComponentType.Tree))
				{
					return Visibility.Visible;
				}
			}
			return Visibility.Collapsed;
		}

		// Token: 0x06000060 RID: 96 RVA: 0x00005B9C File Offset: 0x00003D9C
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
