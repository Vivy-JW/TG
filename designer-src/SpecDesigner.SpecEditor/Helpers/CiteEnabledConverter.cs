using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000009 RID: 9
	public class CiteEnabledConverter : IValueConverter
	{
		// Token: 0x06000059 RID: 89 RVA: 0x00005A70 File Offset: 0x00003C70
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			FormSpecModel formSpecModel = value as FormSpecModel;
			if (formSpecModel == null)
			{
				return false;
			}
			TzpManager tzpManger = SettingManager.Get().GetTzpManger(formSpecModel.Key);
			if (tzpManger == null)
			{
				return false;
			}
			if (tzpManger.IsStandardProgram)
			{
				return false;
			}
			if (tzpManger.SpecificationInfo.CiteSTD == null)
			{
				return false;
			}
			if (formSpecModel.CitedSpec == null)
			{
				return false;
			}
			return true;
		}

		// Token: 0x0600005A RID: 90 RVA: 0x00005AE1 File Offset: 0x00003CE1
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
