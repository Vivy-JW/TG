using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x02000060 RID: 96
	public class ResizeDirectionConverter : IValueConverter
	{
		// Token: 0x060003C2 RID: 962 RVA: 0x0001459C File Offset: 0x0001279C
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = parameter as string;
			XmlElement xmlElement = value as XmlElement;
			bool flag = false;
			if (xmlElement != null)
			{
				switch (xmlElement.GetGeneroResizeDirection())
				{
				case ResizeDirectionEnum.WIDTH:
				{
					string text2;
					if ((text2 = text) != null && (text2 == "Left" || text2 == "Right"))
					{
						flag = true;
					}
					break;
				}
				case ResizeDirectionEnum.HEIGHT:
				{
					string text3;
					if ((text3 = text) != null && (text3 == "Bottom" || text3 == "Top"))
					{
						flag = true;
					}
					break;
				}
				case ResizeDirectionEnum.BOTH:
				{
					string text4;
					switch (text4 = text)
					{
					case "BottomLeft":
					case "Bottom":
					case "BottomRight":
					case "TopLeft":
					case "Top":
					case "TopRight":
					case "Left":
					case "Right":
						flag = true;
						break;
					}
					break;
				}
				case ResizeDirectionEnum.RIGHT:
				{
					string text5;
					if ((text5 = text) != null && text5 == "Right")
					{
						flag = true;
					}
					break;
				}
				case ResizeDirectionEnum.BOTTOM:
				{
					string text6;
					if ((text6 = text) != null && text6 == "Bottom")
					{
						flag = true;
					}
					break;
				}
				}
			}
			return flag;
		}

		// Token: 0x060003C3 RID: 963 RVA: 0x00014730 File Offset: 0x00012930
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
