using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000037 RID: 55
	public class IsCitedVisibilityConverter : IValueConverter
	{
		// Token: 0x06000203 RID: 515 RVA: 0x0000AE88 File Offset: 0x00009088
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			XmlElement xmlElement = value as XmlElement;
			if (xmlElement == null)
			{
				return Visibility.Collapsed;
			}
			if (SettingManager.Get().GetTzpManger(xmlElement.Key).IsStandardProgram)
			{
				return Visibility.Collapsed;
			}
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(xmlElement.Key).SpecificationInfo.FindNodeByName(xmlElement.Name);
			if (formSpecModel != null)
			{
				return formSpecModel.IsCited ? Visibility.Collapsed : Visibility.Visible;
			}
			return Visibility.Visible;
		}

		// Token: 0x06000204 RID: 516 RVA: 0x0000AF00 File Offset: 0x00009100
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
