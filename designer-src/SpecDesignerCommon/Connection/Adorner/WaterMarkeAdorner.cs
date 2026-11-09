using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Documents;
using System.Windows.Media;

namespace SpecDesignerCommon.Connection.Adorner
{
	// Token: 0x0200003D RID: 61
	public class WaterMarkeAdorner : Adorner
	{
		// Token: 0x060001F8 RID: 504 RVA: 0x00008D74 File Offset: 0x00006F74
		public WaterMarkeAdorner(UIElement adornedElement, string label, Style labelStyle)
			: base(adornedElement)
		{
			this.adornerTextBlock = new TextBlock
			{
				Style = labelStyle,
				Text = label
			};
		}

		// Token: 0x060001F9 RID: 505 RVA: 0x00008DA4 File Offset: 0x00006FA4
		public WaterMarkeAdorner(UIElement adornedElement, string label)
			: base(adornedElement)
		{
			this.adornerTextBlock = new TextBlock
			{
				Text = label,
				Margin = new Thickness(2.0)
			};
			this.adornerTextBlock.SetValue(TextBlock.ForegroundProperty, new SolidColorBrush(Colors.Black));
		}

		// Token: 0x060001FA RID: 506 RVA: 0x00008DFA File Offset: 0x00006FFA
		protected override Size MeasureOverride(Size constraint)
		{
			this.adornerTextBlock.Measure(constraint);
			return constraint;
		}

		// Token: 0x060001FB RID: 507 RVA: 0x00008E09 File Offset: 0x00007009
		protected override Size ArrangeOverride(Size finalSize)
		{
			this.adornerTextBlock.Arrange(new Rect(finalSize));
			return finalSize;
		}

		// Token: 0x060001FC RID: 508 RVA: 0x00008E1D File Offset: 0x0000701D
		protected override Visual GetVisualChild(int index)
		{
			return this.adornerTextBlock;
		}

		// Token: 0x17000078 RID: 120
		// (get) Token: 0x060001FD RID: 509 RVA: 0x00008E25 File Offset: 0x00007025
		protected override int VisualChildrenCount
		{
			get
			{
				return 1;
			}
		}

		// Token: 0x040000BC RID: 188
		private readonly TextBlock adornerTextBlock;
	}
}
