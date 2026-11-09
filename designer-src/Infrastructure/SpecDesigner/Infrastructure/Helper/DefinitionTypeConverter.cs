using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x02000031 RID: 49
	public class DefinitionTypeConverter : IValueConverter
	{
		// Token: 0x0600014A RID: 330 RVA: 0x00006ABB File Offset: 0x00004CBB
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			return value.ToString();
		}

		// Token: 0x0600014B RID: 331 RVA: 0x00006AC4 File Offset: 0x00004CC4
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = value.ToString();
			string text2;
			if ((text2 = text) != null)
			{
				if (text2 == "FUNCTION")
				{
					return DefinitionType.FUNCTION;
				}
				if (text2 == "DIALOG")
				{
					return DefinitionType.DIALOG;
				}
				if (text2 == "REPORT")
				{
					return DefinitionType.REPORT;
				}
			}
			return DefinitionType.FUNCTION;
		}
	}
}
