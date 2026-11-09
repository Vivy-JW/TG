using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Windows.Threading;
using SpecDesigner.CodeEditWindow.View;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x0200002F RID: 47
	public class ToolTipBehavior : Behavior<CodeTextEditor>
	{
		// Token: 0x17000047 RID: 71
		// (get) Token: 0x060001D5 RID: 469 RVA: 0x0000F447 File Offset: 0x0000D647
		private PackageKey Key
		{
			get
			{
				return Application.Current.MainWindow.Tag as PackageKey;
			}
		}

		// Token: 0x060001D7 RID: 471 RVA: 0x0000F468 File Offset: 0x0000D668
		protected override void OnAttached()
		{
			base.AssociatedObject.PreviewMouseLeftButtonDown += this.AssociatedObject_PreviewMouseDown;
			base.AssociatedObject.MouseMove += this.AssociatedObject_MouseMove;
			base.AssociatedObject.MouseLeave += this.AssociatedObject_MouseLeave;
			base.AssociatedObject.PreviewKeyDown += this.AssociatedObject_PreviewKeyDown;
			this._tooltip = ToolTipService.GetToolTip(base.AssociatedObject) as ToolTip;
			if (this._tooltip != null)
			{
				this._tooltipTimer = new DispatcherTimer();
				this._tooltipTimer.Interval = new TimeSpan(0, 0, 0, 0, 3000);
				this._tooltipTimer.Tick += this._timer_Tick;
				this._tooltip.IsVisibleChanged += this._tooltip_IsVisibleChanged;
			}
		}

		// Token: 0x060001D8 RID: 472 RVA: 0x0000F541 File Offset: 0x0000D741
		private void _tooltip_IsVisibleChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			if (this._tooltip.IsOpen)
			{
				this._tooltipTimer.Start();
			}
		}

		// Token: 0x060001D9 RID: 473 RVA: 0x0000F55C File Offset: 0x0000D75C
		protected override void OnDetaching()
		{
			base.AssociatedObject.PreviewMouseLeftButtonDown -= this.AssociatedObject_PreviewMouseDown;
			base.AssociatedObject.MouseMove -= this.AssociatedObject_MouseMove;
			base.AssociatedObject.MouseLeave -= this.AssociatedObject_MouseLeave;
			base.AssociatedObject.PreviewKeyDown -= this.AssociatedObject_PreviewKeyDown;
			if (this._tooltipTimer != null)
			{
				this._tooltipTimer.Stop();
				this._tooltipTimer.Tick -= this._timer_Tick;
			}
		}

		// Token: 0x060001DA RID: 474 RVA: 0x0000F5EF File Offset: 0x0000D7EF
		private void AssociatedObject_PreviewMouseDown(object sender, MouseButtonEventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x060001DB RID: 475 RVA: 0x0000F5F7 File Offset: 0x0000D7F7
		private void _timer_Tick(object sender, EventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x060001DC RID: 476 RVA: 0x0000F5FF File Offset: 0x0000D7FF
		private void AssociatedObject_PreviewKeyDown(object sender, KeyEventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x060001DD RID: 477 RVA: 0x0000F607 File Offset: 0x0000D807
		private void CloseTooltip()
		{
			this._tooltip.Content = null;
			this._tooltip.IsOpen = false;
			this._tooltipTimer.Stop();
		}

		// Token: 0x060001DE RID: 478 RVA: 0x0000F62C File Offset: 0x0000D82C
		private void AssociatedObject_MouseMove(object sender, MouseEventArgs e)
		{
			e.Handled = true;
			if (this.Key == null)
			{
				return;
			}
			if (this._tooltip != null)
			{
				this._tooltip.IsOpen = false;
			}
		}

		// Token: 0x060001DF RID: 479 RVA: 0x0000F658 File Offset: 0x0000D858
		private void AssociatedObject_MouseLeave(object sender, MouseEventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x040000C8 RID: 200
		private ToolTip _tooltip;

		// Token: 0x040000C9 RID: 201
		private DispatcherTimer _tooltipTimer;
	}
}
