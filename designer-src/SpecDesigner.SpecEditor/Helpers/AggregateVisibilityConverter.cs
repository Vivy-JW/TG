using System;
using System.Globalization;
using System.Windows;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200003D RID: 61
	public class AggregateVisibilityConverter : ConverterMarkupExtension<AggregateVisibilityConverter>
	{
		// Token: 0x0600018E RID: 398 RVA: 0x0000B1C8 File Offset: 0x000093C8
		public override object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value is XmlElement)
			{
				XmlElement xmlElement = value as XmlElement;
				if (xmlElement.Type != ComponentType.Phantom && xmlElement.Parent != null && xmlElement.Parent.Type == ComponentType.Table)
				{
					return Visibility.Visible;
				}
			}
			return Visibility.Collapsed;
		}

		// Token: 0x0600018F RID: 399 RVA: 0x0000B212 File Offset: 0x00009412
		public override object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
