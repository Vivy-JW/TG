using System;
using System.Collections.Generic;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000030 RID: 48
	public class MultiBooleanCheckedConverter : IValueConverter
	{
		// Token: 0x06000137 RID: 311 RVA: 0x00009DA0 File Offset: 0x00007FA0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			List<XmlElement> list = value as List<XmlElement>;
			string text = parameter as string;
			if (list == null || string.IsNullOrEmpty(text))
			{
				return false;
			}
			this._list = list;
			bool? flag = null;
			for (int i = 0; i < this._list.Count; i++)
			{
				XmlElement xmlElement = this._list[i];
				bool? flag2 = null;
				if (xmlElement.GetAttribute(text) == null)
				{
					return null;
				}
				string attribute = xmlElement.GetAttribute(text);
				if (attribute.Equals(bool.TrueString, StringComparison.CurrentCultureIgnoreCase))
				{
					flag2 = new bool?(true);
				}
				else if (attribute.Equals(bool.FalseString, StringComparison.CurrentCultureIgnoreCase))
				{
					flag2 = new bool?(false);
				}
				if (flag2 != flag && i > 0)
				{
					flag = null;
				}
				else
				{
					flag = flag2;
				}
			}
			return flag;
		}

		// Token: 0x06000138 RID: 312 RVA: 0x00009E9C File Offset: 0x0000809C
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			MultiFormAttributesUndoRedoCommand multiFormAttributesUndoRedoCommand = new MultiFormAttributesUndoRedoCommand(this._list, parameter as string, value.ToString().ToLower());
			multiFormAttributesUndoRedoCommand.Execute();
			return this._list;
		}

		// Token: 0x04000099 RID: 153
		private List<XmlElement> _list;
	}
}
