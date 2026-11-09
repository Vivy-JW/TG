using System;
using System.Globalization;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Media.Imaging;

namespace SpecDesigner.Converters
{
	// Token: 0x0200001D RID: 29
	public sealed class StringToImageConverter : IValueConverter
	{
		// Token: 0x0600022C RID: 556 RVA: 0x0000A7F4 File Offset: 0x000089F4
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			object obj;
			try
			{
				string text = string.Format("/{0}", value);
				obj = new Image
				{
					Source = new BitmapImage(new Uri(text, UriKind.Relative))
				};
			}
			catch (Exception)
			{
				obj = null;
			}
			return obj;
		}

		// Token: 0x0600022D RID: 557 RVA: 0x0000A840 File Offset: 0x00008A40
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
