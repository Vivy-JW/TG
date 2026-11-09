using System;
using System.Globalization;
using System.IO;
using System.Security;
using System.Windows.Data;
using System.Windows.Markup;
using System.Xml;

namespace SpecDesigner.FormEditor.Test
{
	// Token: 0x0200005C RID: 92
	public class StringToXamlConverter : IMultiValueConverter
	{
		// Token: 0x0600038C RID: 908 RVA: 0x0001369C File Offset: 0x0001189C
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if (values == null)
			{
				return null;
			}
			string text = values[0] as string;
			if (text != null)
			{
				string text2 = text;
				if (((string)values[1]).Length > 0)
				{
					values[1] = SecurityElement.Escape(values[1].ToString());
					text2 = text2.Replace((string)values[1], "@RS@" + (string)values[1] + "@RE@");
				}
				text2 = SecurityElement.Escape(text2);
				if (((string)values[1]).Length > 0)
				{
					text2 = text2.Replace("@RS@", "<Run Background=\"Red\">");
					text2 = text2.Replace("@RE@", "</Run>");
				}
				text2 = text2.Replace(Environment.NewLine, "<LineBreak/>");
				string text3 = string.Format("<TextBlock xmlns=\"http://schemas.microsoft.com/winfx/2006/xaml/presentation\" TextWrapping=\"Wrap\" xml:space=\"preserve\">{0}</TextBlock>", text2);
				using (StringReader stringReader = new StringReader(text3))
				{
					using (XmlReader xmlReader = XmlReader.Create(stringReader))
					{
						return XamlReader.Load(xmlReader);
					}
				}
			}
			return null;
		}

		// Token: 0x0600038D RID: 909 RVA: 0x000137B0 File Offset: 0x000119B0
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
