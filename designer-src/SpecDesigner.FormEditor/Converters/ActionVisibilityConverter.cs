using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x02000023 RID: 35
	public class ActionVisibilityConverter : IValueConverter
	{
		// Token: 0x06000128 RID: 296 RVA: 0x00006DFC File Offset: 0x00004FFC
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			SpecActionNode specActionNode = value as SpecActionNode;
			string text = parameter as string;
			if (specActionNode == null || string.IsNullOrEmpty(text))
			{
				return Visibility.Collapsed;
			}
			if ((specActionNode.Status & SpecStatus.DELETE) != SpecStatus.DELETE && !specActionNode.IsActionDefaults && specActionNode.IsContainsType(text))
			{
				return Visibility.Visible;
			}
			return Visibility.Collapsed;
		}

		// Token: 0x06000129 RID: 297 RVA: 0x00006E52 File Offset: 0x00005052
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
