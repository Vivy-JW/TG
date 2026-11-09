using System;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using System.Windows.Media;
using SpecDesigner.FormEditor.Helpers;
using SpecDesignerCommon;

namespace SpecDesigner.FormEditor.Converters
{
	// Token: 0x02000051 RID: 81
	public class TabIndexBackgroundConverter : IMultiValueConverter
	{
		// Token: 0x06000346 RID: 838 RVA: 0x000120EC File Offset: 0x000102EC
		public TabIndexBackgroundConverter()
		{
			this._emptyBrush = new LinearGradientBrush(Color.FromArgb(byte.MaxValue, 171, 171, 171), Color.FromArgb(byte.MaxValue, 219, 219, 219), new Point(0.0, 0.0), new Point(0.0, 0.9));
			this._lessBrush = new LinearGradientBrush(Color.FromArgb(byte.MaxValue, 35, 132, 16), Color.FromArgb(byte.MaxValue, 152, 221, 136), new Point(0.0, 0.0), new Point(0.0, 0.9));
			this._equalBrush = new LinearGradientBrush(Color.FromArgb(byte.MaxValue, byte.MaxValue, 119, 107), Color.FromArgb(byte.MaxValue, byte.MaxValue, 168, 158), new Point(0.0, 0.0), new Point(0.0, 0.9));
			this._greaterBrush = new LinearGradientBrush(Color.FromArgb(byte.MaxValue, 111, 175, 235), Color.FromArgb(byte.MaxValue, 159, 200, 239), new Point(0.0, 0.0), new Point(0.0, 0.9));
		}

		// Token: 0x06000347 RID: 839 RVA: 0x0001229C File Offset: 0x0001049C
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			int num = 0;
			if (int.TryParse(values[0].ToString(), out num))
			{
				if (num == 0)
				{
					return this._emptyBrush;
				}
				PackageKey packageKey = values[1] as PackageKey;
				int currentIndex = ComponentTabIndexService.Get(packageKey).CurrentIndex;
				if (num < currentIndex)
				{
					return this._lessBrush;
				}
				if (num == currentIndex)
				{
					return this._equalBrush;
				}
				if (num > currentIndex)
				{
					return this._greaterBrush;
				}
			}
			return this._emptyBrush;
		}

		// Token: 0x06000348 RID: 840 RVA: 0x00012302 File Offset: 0x00010502
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}

		// Token: 0x040001A9 RID: 425
		private LinearGradientBrush _emptyBrush;

		// Token: 0x040001AA RID: 426
		private LinearGradientBrush _lessBrush;

		// Token: 0x040001AB RID: 427
		private LinearGradientBrush _equalBrush;

		// Token: 0x040001AC RID: 428
		private LinearGradientBrush _greaterBrush;
	}
}
