using System;
using System.Globalization;
using System.IO;
using System.Windows;
using System.Windows.Data;
using System.Windows.Media.Imaging;
using System.Windows.Resources;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000050 RID: 80
	public class ButtonEditImageConverter : IValueConverter
	{
		// Token: 0x06000343 RID: 835 RVA: 0x00012058 File Offset: 0x00010258
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (value == null)
			{
				return null;
			}
			if ("" == value.ToString())
			{
				return null;
			}
			string fileName = Path.GetFileName(value.ToString());
			StreamResourceInfo streamResourceInfo = null;
			try
			{
				streamResourceInfo = Application.GetResourceStream(new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Images/" + fileName));
			}
			catch
			{
				return null;
			}
			BitmapImage bitmapImage = new BitmapImage();
			bitmapImage.BeginInit();
			bitmapImage.StreamSource = streamResourceInfo.Stream;
			bitmapImage.EndInit();
			return bitmapImage;
		}

		// Token: 0x06000344 RID: 836 RVA: 0x000120DC File Offset: 0x000102DC
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
