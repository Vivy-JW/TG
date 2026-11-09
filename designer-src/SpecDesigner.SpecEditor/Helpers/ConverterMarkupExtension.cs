using System;
using System.Globalization;
using System.Windows.Data;
using System.Windows.Markup;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000003 RID: 3
	public abstract class ConverterMarkupExtension<T> : MarkupExtension, IValueConverter where T : class, new()
	{
		// Token: 0x06000006 RID: 6 RVA: 0x00002139 File Offset: 0x00000339
		public override object ProvideValue(IServiceProvider serviceProvider)
		{
			if (ConverterMarkupExtension<T>.m_converter == null)
			{
				ConverterMarkupExtension<T>.m_converter = new T();
			}
			return ConverterMarkupExtension<T>.m_converter;
		}

		// Token: 0x06000007 RID: 7
		public abstract object Convert(object value, Type targetType, object parameter, CultureInfo culture);

		// Token: 0x06000008 RID: 8
		public abstract object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture);

		// Token: 0x04000003 RID: 3
		private static T m_converter = default(T);
	}
}
