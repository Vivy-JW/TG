using System;
using System.Globalization;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows.Data;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200000C RID: 12
	internal class StringUpConverter : IValueConverter
	{
		// Token: 0x06000062 RID: 98 RVA: 0x00005BD8 File Offset: 0x00003DD8
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return "";
			}
			string p1 = "^" + value.ToString() + "(?=\\.)";
			XElement xelement = (from d in SettingManager.Get().Info_DataTypes.Descendants("dataType")
				where Regex.IsMatch(d.Attribute("name").Value, p1)
				select d).FirstOrDefault<XElement>();
			if (xelement != null)
			{
				string text = "(?<desc>^\\d+.\\S+):";
				string value2 = xelement.Attribute("name").Value;
				Match match = Regex.Match(value2, text);
				return match.Groups["desc"].Value;
			}
			return null;
		}

		// Token: 0x06000063 RID: 99 RVA: 0x00005C80 File Offset: 0x00003E80
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
