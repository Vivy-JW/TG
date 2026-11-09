using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace SpecDesignerCommon
{
	// Token: 0x02000011 RID: 17
	public class Watermark : TextBlock
	{
		// Token: 0x06000077 RID: 119 RVA: 0x00003CA4 File Offset: 0x00001EA4
		public Watermark()
		{
			base.IsHitTestVisible = false;
			base.VerticalAlignment = VerticalAlignment.Bottom;
			base.HorizontalAlignment = HorizontalAlignment.Right;
			base.Text = Application.Current.FindResource("Message_NotBooking") as string;
			base.Foreground = Brushes.Red;
			base.FontFamily = new FontFamily("Microsoft JhengHei");
			base.FontWeight = FontWeights.ExtraBold;
			base.FontSize = 70.0;
			base.Opacity = 0.3;
			Panel.SetZIndex(this, int.MaxValue);
		}
	}
}
