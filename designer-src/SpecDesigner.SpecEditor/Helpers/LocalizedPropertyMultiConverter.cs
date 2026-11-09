using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000021 RID: 33
	internal class LocalizedPropertyMultiConverter : IMultiValueConverter
	{
		// Token: 0x060000FE RID: 254 RVA: 0x0000946C File Offset: 0x0000766C
		public object Convert(object[] value, Type targetType, object parameter, CultureInfo culture)
		{
			ResourceDictionary resourceDictionary = Application.LoadComponent(new Uri("/SpecDesignerCommon;component/Langs/en-US.xaml", UriKind.RelativeOrAbsolute)) as ResourceDictionary;
			string text = parameter.ToString();
			string text2 = resourceDictionary[text].ToString();
			bool flag2;
			bool flag = bool.TryParse(value[0].ToString(), out flag2);
			if (flag && flag2)
			{
				text2 = Application.Current.FindResource(text).ToString();
			}
			return text2;
		}

		// Token: 0x060000FF RID: 255 RVA: 0x000094CD File Offset: 0x000076CD
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
