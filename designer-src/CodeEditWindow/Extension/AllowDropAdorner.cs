using System;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Media;

namespace SpecDesigner.CodeEditWindow.Extension
{
	// Token: 0x0200001F RID: 31
	internal class AllowDropAdorner : Adorner
	{
		// Token: 0x0600011B RID: 283 RVA: 0x0000B284 File Offset: 0x00009484
		public AllowDropAdorner(UIElement adornedElement)
			: base(adornedElement)
		{
			base.Visibility = Visibility.Collapsed;
			this.adornerLayer = AdornerLayer.GetAdornerLayer(adornedElement);
			this.adornerLayer.Add(this);
		}

		// Token: 0x0600011C RID: 284 RVA: 0x0000B2EC File Offset: 0x000094EC
		protected override void OnRender(DrawingContext drawingContext)
		{
			if (base.Visibility != Visibility.Visible)
			{
				return;
			}
			Rect rect = new Rect(base.AdornedElement.RenderSize);
			drawingContext.DrawLine(this._pen, new Point(rect.Left - (double)this.OFFSET, rect.Height + (double)this.OFFSET), new Point(rect.Right + (double)this.OFFSET, rect.Height + (double)this.OFFSET));
		}

		// Token: 0x0600011D RID: 285 RVA: 0x0000B366 File Offset: 0x00009566
		public void Remove()
		{
			base.Visibility = Visibility.Collapsed;
		}

		// Token: 0x0600011E RID: 286 RVA: 0x0000B36F File Offset: 0x0000956F
		public void Update()
		{
			base.Visibility = Visibility.Visible;
		}

		// Token: 0x04000088 RID: 136
		private AdornerLayer adornerLayer;

		// Token: 0x04000089 RID: 137
		private Pen _pen = new Pen
		{
			Brush = new SolidColorBrush(Colors.DarkGray),
			Thickness = 2.0
		};

		// Token: 0x0400008A RID: 138
		private readonly int OFFSET = 2;
	}
}
