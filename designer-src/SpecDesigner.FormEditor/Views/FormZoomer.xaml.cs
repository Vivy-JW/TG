using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Media;
using System.Windows.Media.Imaging;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000032 RID: 50
	public partial class FormZoomer : Window, IDisposable
	{
		// Token: 0x060001CE RID: 462 RVA: 0x00009508 File Offset: 0x00007708
		public FormZoomer(ScrollViewer scroll)
		{
			this.InitializeComponent();
			this.scroll = scroll;
			base.Loaded += this.FormZoomer_Loaded;
			this.attachDnDEvent(this.zoom);
		}

		// Token: 0x060001CF RID: 463 RVA: 0x00009561 File Offset: 0x00007761
		private void attachDnDEvent(Canvas zoom)
		{
			zoom.MouseLeftButtonDown += this.zoom_MouseLeftButtonDown;
			zoom.MouseMove += this.zoom_MouseMove;
			zoom.MouseLeftButtonUp += this.zoom_MouseLeftButtonUp;
		}

		// Token: 0x060001D0 RID: 464 RVA: 0x00009599 File Offset: 0x00007799
		private void zoom_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			this.isMouseLeftButtonDown = false;
			this.zoom.ReleaseMouseCapture();
			this._offset = default(Point);
		}

		// Token: 0x060001D1 RID: 465 RVA: 0x000095BC File Offset: 0x000077BC
		private void zoom_MouseMove(object sender, MouseEventArgs e)
		{
			if (this.isMouseLeftButtonDown)
			{
				Point position = e.GetPosition(this.layoutRoot);
				double num = Canvas.GetLeft(this.zoom) + (position.X - this._offset.X);
				double num2 = Canvas.GetTop(this.zoom) + (position.Y - this._offset.Y);
				Canvas.SetTop(this.zoom, num2);
				Canvas.SetLeft(this.zoom, num);
				this._offset = position;
				this.syncScrollViewer();
			}
		}

		// Token: 0x060001D2 RID: 466 RVA: 0x00009643 File Offset: 0x00007843
		private void zoom_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			this.isMouseLeftButtonDown = true;
			this.zoom.CaptureMouse();
			this._offset = e.GetPosition(this.layoutRoot);
		}

		// Token: 0x060001D3 RID: 467 RVA: 0x0000966C File Offset: 0x0000786C
		private void syncScrollViewer()
		{
			double top = Canvas.GetTop(this.zoom);
			double left = Canvas.GetLeft(this.zoom);
			this.scroll.ScrollToVerticalOffset(top / this.scale);
			this.scroll.ScrollToHorizontalOffset(left / this.scale);
		}

		// Token: 0x060001D4 RID: 468 RVA: 0x000096B7 File Offset: 0x000078B7
		private void FormZoomer_Loaded(object sender, RoutedEventArgs e)
		{
			this.scroll.ScrollChanged += this.scroll_ScrollChanged;
			this.init();
			this.updatePosition();
			base.SizeToContent = SizeToContent.WidthAndHeight;
		}

		// Token: 0x060001D5 RID: 469 RVA: 0x000096E4 File Offset: 0x000078E4
		private void updatePosition()
		{
			this.zoom.Width = this.scroll.ViewportWidth * this.scale;
			this.zoom.Height = this.scroll.ViewportHeight * this.scale;
			Canvas.SetTop(this.zoom, this.scroll.VerticalOffset * this.scale);
			Canvas.SetLeft(this.zoom, this.scroll.HorizontalOffset * this.scale);
		}

		// Token: 0x060001D6 RID: 470 RVA: 0x00009768 File Offset: 0x00007968
		private void init()
		{
			double num;
			if (this.scroll.ExtentHeight > this.scroll.ExtentWidth)
			{
				num = this.scroll.ExtentWidth;
			}
			else
			{
				num = this.scroll.ExtentHeight;
			}
			this.scale = 150.0 / num;
			this.layoutRoot.Width = this.scroll.ExtentWidth * this.scale;
			this.layoutRoot.Height = this.scroll.ExtentHeight * this.scale;
			this.renderThumb();
		}

		// Token: 0x060001D7 RID: 471 RVA: 0x00009804 File Offset: 0x00007A04
		private void scroll_ScrollChanged(object sender, ScrollChangedEventArgs e)
		{
			if (e.ExtentHeightChange + e.ExtentWidthChange > 0.0)
			{
				this.init();
			}
			TextBox textBox = this.tbBox;
			textBox.Text = textBox.Text + "ExtentHeightChange was " + e.ExtentHeightChange.ToString();
			TextBox textBox2 = this.tbBox;
			textBox2.Text = textBox2.Text + "ExtentWidthChange was " + e.ExtentWidthChange.ToString();
			TextBox textBox3 = this.tbBox;
			textBox3.Text = textBox3.Text + "\r\n ExtentHeight is now " + this.scroll.ExtentHeight.ToString();
			TextBox textBox4 = this.tbBox;
			textBox4.Text = textBox4.Text + "\r\n ExtentWidth is now " + this.scroll.ExtentWidth.ToString();
			TextBox textBox5 = this.tbBox;
			textBox5.Text = textBox5.Text + "\r\n HorizontalOffset is now " + this.scroll.HorizontalOffset.ToString();
			TextBox textBox6 = this.tbBox;
			textBox6.Text = textBox6.Text + "\r\n VerticalOffset is now " + this.scroll.VerticalOffset.ToString();
			TextBox textBox7 = this.tbBox;
			textBox7.Text = textBox7.Text + "\r\n ViewportHeight is now " + this.scroll.ViewportHeight.ToString();
			TextBox textBox8 = this.tbBox;
			textBox8.Text = textBox8.Text + "\r\n ViewportWidth is now " + this.scroll.ViewportWidth.ToString();
			this.updatePosition();
		}

		// Token: 0x060001D8 RID: 472 RVA: 0x000099A0 File Offset: 0x00007BA0
		private static BitmapSource renderVisualScreen(FrameworkElement target, double scale)
		{
			if (target == null)
			{
				return null;
			}
			double height = target.RenderSize.Height;
			double width = target.RenderSize.Width;
			double num = height * scale;
			double num2 = width * scale;
			RenderTargetBitmap renderTargetBitmap = new RenderTargetBitmap((int)num2, (int)num, 96.0, 96.0, PixelFormats.Pbgra32);
			DrawingVisual drawingVisual = new DrawingVisual();
			using (DrawingContext drawingContext = drawingVisual.RenderOpen())
			{
				drawingContext.PushTransform(new ScaleTransform(scale, scale));
				VisualBrush visualBrush = new VisualBrush(target);
				drawingContext.DrawRectangle(visualBrush, null, new Rect(new Size(width, height)));
			}
			renderTargetBitmap.Render(drawingVisual);
			renderTargetBitmap.Freeze();
			return renderTargetBitmap;
		}

		// Token: 0x060001D9 RID: 473 RVA: 0x00009A68 File Offset: 0x00007C68
		private void refresh_Click(object sender, RoutedEventArgs e)
		{
			this.renderThumb();
		}

		// Token: 0x060001DA RID: 474 RVA: 0x00009A70 File Offset: 0x00007C70
		private void renderThumb()
		{
			FrameworkElement frameworkElement = this.scroll.Content as FrameworkElement;
			this.bgImage.Source = FormZoomer.renderVisualScreen(frameworkElement.FindName("main_panel") as FrameworkElement, this.scale);
		}

		// Token: 0x060001DB RID: 475 RVA: 0x00009AB4 File Offset: 0x00007CB4
		public void Dispose()
		{
			this.scroll.ScrollChanged -= this.scroll_ScrollChanged;
			base.Loaded -= this.FormZoomer_Loaded;
			this.zoom.MouseLeftButtonDown -= this.zoom_MouseLeftButtonDown;
			this.zoom.MouseMove -= this.zoom_MouseMove;
			this.zoom.MouseLeftButtonUp -= this.zoom_MouseLeftButtonUp;
		}

		// Token: 0x04000103 RID: 259
		private ScrollViewer scroll;

		// Token: 0x04000104 RID: 260
		private double scale = 1.0;

		// Token: 0x04000105 RID: 261
		private bool isMouseLeftButtonDown;

		// Token: 0x04000106 RID: 262
		private Point _offset = default(Point);
	}
}
