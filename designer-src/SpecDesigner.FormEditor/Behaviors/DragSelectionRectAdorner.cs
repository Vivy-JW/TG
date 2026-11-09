using System;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Media;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x02000056 RID: 86
	public class DragSelectionRectAdorner : Adorner
	{
		// Token: 0x0600036B RID: 875 RVA: 0x00012D1C File Offset: 0x00010F1C
		public DragSelectionRectAdorner(UIElement element)
			: base(element)
		{
			this._pen = new Pen
			{
				Brush = new SolidColorBrush(Colors.Black),
				Thickness = 1.0
			};
			this._rect = default(Rect);
		}

		// Token: 0x0600036C RID: 876 RVA: 0x00012D80 File Offset: 0x00010F80
		protected override void OnRender(DrawingContext drawingContext)
		{
			this._rect.X = Math.Min(this.StartPoint.X, this.EndPoint.X);
			this._rect.Y = Math.Min(this.StartPoint.Y, this.EndPoint.Y);
			this._rect.Width = Math.Max(Math.Max(this.StartPoint.X, this.EndPoint.X) - this._rect.Left, 0.0);
			this._rect.Height = Math.Max(Math.Max(this.StartPoint.Y, this.EndPoint.Y) - this._rect.Top, 0.0);
			drawingContext.DrawRectangle(null, this._pen, this._rect);
		}

		// Token: 0x0600036D RID: 877 RVA: 0x00012E6E File Offset: 0x0001106E
		public void UpdateRectPoint(Point p)
		{
			this.EndPoint = p;
			base.InvalidateVisual();
		}

		// Token: 0x040001BD RID: 445
		public Point StartPoint = default(Point);

		// Token: 0x040001BE RID: 446
		public Point EndPoint = default(Point);

		// Token: 0x040001BF RID: 447
		private Rect _rect;

		// Token: 0x040001C0 RID: 448
		private Pen _pen;
	}
}
