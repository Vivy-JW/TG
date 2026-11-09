using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200002F RID: 47
	internal class TextAttributeVisibilityConverter : IMultiValueConverter
	{
		// Token: 0x06000134 RID: 308 RVA: 0x00009D28 File Offset: 0x00007F28
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			if (!(values[1] is ComponentType) || !(values[2] is ComponentType))
			{
				return Visibility.Collapsed;
			}
			if (values[0] == null)
			{
				return Visibility.Collapsed;
			}
			ComponentType componentType = (ComponentType)values[1];
			ComponentType componentType2 = (ComponentType)values[2];
			if ((componentType2 == ComponentType.Tree || componentType2 == ComponentType.Table) && componentType != ComponentType.Phantom && componentType != ComponentType.CheckBox)
			{
				return Visibility.Collapsed;
			}
			return Visibility.Visible;
		}

		// Token: 0x06000135 RID: 309 RVA: 0x00009D8F File Offset: 0x00007F8F
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
