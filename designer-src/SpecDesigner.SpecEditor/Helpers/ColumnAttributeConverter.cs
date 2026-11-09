using System;
using System.Globalization;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows.Data;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000022 RID: 34
	internal class ColumnAttributeConverter : IValueConverter
	{
		// Token: 0x06000101 RID: 257 RVA: 0x00009508 File Offset: 0x00007708
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
				string value2 = xelement.Attribute("name").Value;
				string[] array = value2.Split(new char[] { ':' });
				if (array.Count<string>() > 0)
				{
					return array[array.Count<string>() - 1];
				}
			}
			return null;
		}

		// Token: 0x06000102 RID: 258 RVA: 0x000095B4 File Offset: 0x000077B4
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
