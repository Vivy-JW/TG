using System;
using System.Collections.Generic;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200002E RID: 46
	public class MutiComboValueConveter : IValueConverter
	{
		// Token: 0x06000131 RID: 305 RVA: 0x00009C6C File Offset: 0x00007E6C
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			List<XmlElement> list = value as List<XmlElement>;
			string text = parameter as string;
			if (list == null || string.IsNullOrEmpty(text))
			{
				return false;
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
					text2 = "[[RE][SET]]";
				}
				else
				{
					text2 = attribute;
				}
			}
			return text2;
		}

		// Token: 0x06000132 RID: 306 RVA: 0x00009CEC File Offset: 0x00007EEC
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			MultiFormAttributesUndoRedoCommand multiFormAttributesUndoRedoCommand = new MultiFormAttributesUndoRedoCommand(this._list, parameter as string, value.ToString());
			multiFormAttributesUndoRedoCommand.Execute();
			return this._list;
		}

		// Token: 0x04000098 RID: 152
		private List<XmlElement> _list;
	}
}
