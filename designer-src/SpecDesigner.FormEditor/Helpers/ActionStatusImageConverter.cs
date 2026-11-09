using System;
using System.Globalization;
using System.Windows.Data;
using System.Windows.Media.Imaging;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x02000054 RID: 84
	public class ActionStatusImageConverter : IValueConverter
	{
		// Token: 0x0600035F RID: 863 RVA: 0x000128CC File Offset: 0x00010ACC
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (!(bool)value)
			{
				return ActionStatusImageConverter.enableImage;
			}
			return ActionStatusImageConverter.disableImage;
		}

		// Token: 0x06000360 RID: 864 RVA: 0x000128EE File Offset: 0x00010AEE
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}

		// Token: 0x040001B4 RID: 436
		private static BitmapImage enableImage = new BitmapImage(new Uri("/SpecDesigner.FormEditor;component/Images/idea.png", UriKind.Relative));

		// Token: 0x040001B5 RID: 437
		private static BitmapImage disableImage = new BitmapImage(new Uri("/SpecDesigner.FormEditor;component/Images/idea_disable.png", UriKind.Relative));
	}
}
