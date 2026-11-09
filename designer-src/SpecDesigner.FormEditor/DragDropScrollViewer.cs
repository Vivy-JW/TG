using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Threading;

namespace SpecDesigner.FormEditor
{
	// Token: 0x02000009 RID: 9
	public class DragDropScrollViewer : ScrollViewer, IDisposable
	{
		// Token: 0x06000025 RID: 37 RVA: 0x00002890 File Offset: 0x00000A90
		public void OnMouseMove()
		{
			if (base.IsVisible)
			{
				Point mousePosition = MouseUtilities.GetMousePosition(this);
				if ((mousePosition.Y < DragDropScrollViewer.s_dragMarginHeight || mousePosition.Y > base.RenderSize.Height - DragDropScrollViewer.s_dragMarginHeight || mousePosition.X < DragDropScrollViewer.s_dragMarginWidth || mousePosition.X > base.RenderSize.Width - DragDropScrollViewer.s_dragMarginWidth) && this._dragScrollTimer == null)
				{
					this._dragVelocity = DragDropScrollViewer.s_dragInitialVelocity;
					this._dragScrollTimer = new DispatcherTimer();
					this._dragScrollTimer.Tick += this.TickDragScroll;
					this._dragScrollTimer.Interval = new TimeSpan(0, 0, 0, 0, (int)DragDropScrollViewer.s_dragInterval);
					this._dragScrollTimer.Start();
				}
			}
		}

		// Token: 0x06000026 RID: 38 RVA: 0x00002960 File Offset: 0x00000B60
		private void TickDragScroll(object sender, EventArgs e)
		{
			bool flag = true;
			if (base.IsLoaded)
			{
				Rect rect = new Rect(new Point(0.0, 0.0), base.RenderSize);
				Point mousePosition = MouseUtilities.GetMousePosition(this);
				rect.Width += base.HorizontalOffset;
				rect.Height += base.VerticalOffset;
				if (Mouse.LeftButton == MouseButtonState.Pressed)
				{
					if (mousePosition.Y < DragDropScrollViewer.s_dragMarginHeight)
					{
						this.DragScroll(DragDropScrollViewer.DragDirection.Up);
						flag = false;
					}
					else if (mousePosition.Y > base.RenderSize.Height - DragDropScrollViewer.s_dragMarginHeight)
					{
						this.DragScroll(DragDropScrollViewer.DragDirection.Down);
						flag = false;
					}
					if (mousePosition.X < DragDropScrollViewer.s_dragMarginWidth)
					{
						this.DragScroll(DragDropScrollViewer.DragDirection.Left);
						flag = false;
					}
					else if (mousePosition.X > base.RenderSize.Width - DragDropScrollViewer.s_dragMarginWidth)
					{
						this.DragScroll(DragDropScrollViewer.DragDirection.Right);
						flag = false;
					}
				}
			}
			if (flag)
			{
				this.CancelDrag();
			}
		}

		// Token: 0x06000027 RID: 39 RVA: 0x00002A5F File Offset: 0x00000C5F
		private void CancelDrag()
		{
			if (this._dragScrollTimer != null)
			{
				this._dragScrollTimer.Tick -= this.TickDragScroll;
				this._dragScrollTimer.Stop();
				this._dragScrollTimer = null;
			}
		}

		// Token: 0x06000028 RID: 40 RVA: 0x00002A94 File Offset: 0x00000C94
		private void DragScroll(DragDropScrollViewer.DragDirection direction)
		{
			switch (direction)
			{
			case DragDropScrollViewer.DragDirection.Down:
			case DragDropScrollViewer.DragDirection.Up:
			{
				bool flag = direction == DragDropScrollViewer.DragDirection.Up;
				double num = Math.Max(0.0, base.VerticalOffset + (flag ? (-(this._dragVelocity * DragDropScrollViewer.s_dragInterval)) : (this._dragVelocity * DragDropScrollViewer.s_dragInterval)));
				base.ScrollToVerticalOffset(num);
				break;
			}
			case DragDropScrollViewer.DragDirection.Left:
			case DragDropScrollViewer.DragDirection.Right:
			{
				bool flag2 = direction == DragDropScrollViewer.DragDirection.Left;
				double num = Math.Max(0.0, base.HorizontalOffset + (flag2 ? (-(this._dragVelocity * DragDropScrollViewer.s_dragInterval)) : (this._dragVelocity * DragDropScrollViewer.s_dragInterval)));
				base.ScrollToHorizontalOffset(num);
				break;
			}
			}
			this._dragVelocity = Math.Min(DragDropScrollViewer.s_dragMaxVelocity, this._dragVelocity + DragDropScrollViewer.s_dragAcceleration * DragDropScrollViewer.s_dragInterval);
		}

		// Token: 0x06000029 RID: 41 RVA: 0x00002B62 File Offset: 0x00000D62
		public void Dispose()
		{
		}

		// Token: 0x04000015 RID: 21
		private static readonly double s_dragInterval = 50.0;

		// Token: 0x04000016 RID: 22
		private static readonly double s_dragAcceleration = 0.0005;

		// Token: 0x04000017 RID: 23
		private static readonly double s_dragMaxVelocity = 2.0;

		// Token: 0x04000018 RID: 24
		private static readonly double s_dragInitialVelocity = 0.05;

		// Token: 0x04000019 RID: 25
		private static double s_dragMarginHeight = 40.0;

		// Token: 0x0400001A RID: 26
		private static double s_dragMarginWidth = 60.0;

		// Token: 0x0400001B RID: 27
		private DispatcherTimer _dragScrollTimer;

		// Token: 0x0400001C RID: 28
		private double _dragVelocity;

		// Token: 0x0200000A RID: 10
		private enum DragDirection
		{
			// Token: 0x0400001E RID: 30
			Down,
			// Token: 0x0400001F RID: 31
			Up,
			// Token: 0x04000020 RID: 32
			Left,
			// Token: 0x04000021 RID: 33
			Right
		}
	}
}
