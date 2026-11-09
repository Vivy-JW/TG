using System;
using System.Globalization;
using System.Windows.Data;
using System.Xml;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x0200003D RID: 61
	public class AggregateBackgroundConverter : IValueConverter
	{
		// Token: 0x06000226 RID: 550 RVA: 0x0000BCC0 File Offset: 0x00009EC0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (!(value is XmlElement))
			{
				return false;
			}
			XmlElement xmlElement = value as XmlElement;
			string value2 = xmlElement[parameter.ToString()].Value;
			string text;
			if ((text = value2.ToLower()) != null && text == "true")
			{
				return true;
			}
			return false;
		}

		// Token: 0x06000227 RID: 551 RVA: 0x0000BD19 File Offset: 0x00009F19
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
