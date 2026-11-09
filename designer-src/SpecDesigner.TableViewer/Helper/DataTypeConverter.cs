using System;
using System.Globalization;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows.Data;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.TableViewer.Helper
{
	// Token: 0x02000003 RID: 3
	public class DataTypeConverter : IValueConverter
	{
		// Token: 0x06000016 RID: 22 RVA: 0x0000294C File Offset: 0x00000B4C
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return "";
			}
			Regex match = new Regex("^" + value.ToString() + "(?=\\.)");
			XElement xelement = (from d in SettingManager.Get().Info_DataTypes.Descendants("dataType")
				where match.IsMatch(d.Attribute("name").Value)
				select d).FirstOrDefault<XElement>();
			if (xelement != null)
			{
				Regex regex = new Regex("(?:\\.)(?<name>.*)(?::)");
				if (regex.IsMatch(xelement.Attribute("name").Value))
				{
					Match match2 = regex.Match(xelement.Attribute("name").Value);
					return match2.Groups["name"].Value;
				}
			}
			return null;
		}

		// Token: 0x06000017 RID: 23 RVA: 0x00002A16 File Offset: 0x00000C16
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
