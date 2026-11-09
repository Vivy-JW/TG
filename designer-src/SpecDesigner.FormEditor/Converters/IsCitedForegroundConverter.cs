using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x02000053 RID: 83
	public class IsCitedForegroundConverter : IValueConverter
	{
		// Token: 0x0600035C RID: 860 RVA: 0x0001284C File Offset: 0x00010A4C
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			XmlElement xmlElement = value as XmlElement;
			if (xmlElement == null)
			{
				return true;
			}
			if (SettingManager.Get().GetTzpManger(xmlElement.Key).IsStandardProgram)
			{
				return true;
			}
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(xmlElement.Key).SpecificationInfo.FindNodeByName(xmlElement.Name);
			return formSpecModel != null && formSpecModel.IsCited;
		}

		// Token: 0x0600035D RID: 861 RVA: 0x000128BA File Offset: 0x00010ABA
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
