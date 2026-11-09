using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using System.Windows.Media.Imaging;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x0200008C RID: 140
	public class MessageBoxImageConverter : IValueConverter
	{
		// Token: 0x060005C3 RID: 1475 RVA: 0x0001A5B0 File Offset: 0x000187B0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			MessageBoxImage messageBoxImage = (MessageBoxImage)value;
			MessageBoxImage messageBoxImage2 = messageBoxImage;
			if (messageBoxImage2 <= MessageBoxImage.Question)
			{
				if (messageBoxImage2 == MessageBoxImage.Hand)
				{
					return new BitmapImage(new Uri("/SpecDesignerCommon;component/Images/Error.png", UriKind.Relative));
				}
				if (messageBoxImage2 == MessageBoxImage.Question)
				{
					return new BitmapImage(new Uri("/SpecDesignerCommon;component/Images/Question.png", UriKind.Relative));
				}
			}
			else
			{
				if (messageBoxImage2 == MessageBoxImage.Exclamation)
				{
					return new BitmapImage(new Uri("/SpecDesignerCommon;component/Images/Warning.png", UriKind.Relative));
				}
				if (messageBoxImage2 == MessageBoxImage.Asterisk)
				{
					return new BitmapImage(new Uri("/SpecDesignerCommon;component/Images/Information.png", UriKind.Relative));
				}
			}
			return null;
		}

		// Token: 0x060005C4 RID: 1476 RVA: 0x0001A626 File Offset: 0x00018826
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
