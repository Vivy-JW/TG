using System;
using System.Collections.Generic;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200003F RID: 63
	public class MutiTextValueConveter : IValueConverter
	{
		// Token: 0x06000193 RID: 403 RVA: 0x0000B234 File Offset: 0x00009434
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			List<XmlElement> list = value as List<XmlElement>;
			string text = parameter as string;
			if (list == null || string.IsNullOrEmpty(text))
			{
				return "";
			}
			this._list = list;
			string text2 = string.Empty;
			for (int i = 0; i < list.Count; i++)
			{
				XmlElement xmlElement = list[i];
				string attribute = xmlElement.GetAttribute(text);
				if (attribute == null)
				{
					return null;
				}
				if (attribute != text2 && i > 0)
				{
					text2 = "";
				}
				else
				{
					text2 = attribute;
				}
			}
			return text2;
		}

		// Token: 0x06000194 RID: 404 RVA: 0x0000B2B4 File Offset: 0x000094B4
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			MultiFormAttributesUndoRedoCommand multiFormAttributesUndoRedoCommand = new MultiFormAttributesUndoRedoCommand(this._list, parameter as string, value.ToString());
			multiFormAttributesUndoRedoCommand.Execute();
			return this._list;
		}

		// Token: 0x040000BA RID: 186
		private List<XmlElement> _list;
	}
}
