using System;
using System.Globalization;
using System.Windows;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200004A RID: 74
	public class RepeatVisibilityConverter : ConverterMarkupExtension<RepeatVisibilityConverter>
	{
		// Token: 0x060001CD RID: 461 RVA: 0x0000C774 File Offset: 0x0000A974
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value is XmlElement)
			{
				XmlElement xmlElement = value as XmlElement;
				if (xmlElement.Parent != null && xmlElement.Parent.Type == ComponentType.ScrollGrid)
				{
					return (xmlElement.GetAttribute("repeat") != null) ? Visibility.Visible : Visibility.Collapsed;
				}
			}
			return Visibility.Collapsed;
		}

		// Token: 0x060001CE RID: 462 RVA: 0x0000C7C3 File Offset: 0x0000A9C3
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
