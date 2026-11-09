using System;
using System.Globalization;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows.Data;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000011 RID: 17
	internal class ColumnTypeConverter : IValueConverter
	{
		// Token: 0x0600006F RID: 111 RVA: 0x0000655C File Offset: 0x0000475C
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return "";
			}
			string pattern = "^" + value.ToString() + "(?=\\.)";
			XElement xelement = (from d in SettingManager.Get().Info_DataTypes.Descendants("dataType")
				where Regex.IsMatch(d.Attribute("name").Value, pattern)
				select d).FirstOrDefault<XElement>();
			if (xelement != null)
			{
				string text = "(?<desc>^" + value.ToString() + ".\\S+):";
				string value2 = xelement.Attribute("name").Value;
				Match match = Regex.Match(value2, text);
				return match.Groups["desc"].Value;
			}
			return null;
		}

		// Token: 0x06000070 RID: 112 RVA: 0x00006614 File Offset: 0x00004814
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
