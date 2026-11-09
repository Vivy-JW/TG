using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Windows.Threading;
using ICSharpCode.AvalonEdit;
using ICSharpCode.AvalonEdit.Document;
using ICSharpCode.AvalonEdit.Extension;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000025 RID: 37
	public class ShowHintBehavior : Behavior<TextEditor>
	{
		// Token: 0x17000026 RID: 38
		// (get) Token: 0x06000147 RID: 327 RVA: 0x0000C338 File Offset: 0x0000A538
		private PackageKey Key
		{
			get
			{
				return Application.Current.MainWindow.Tag as PackageKey;
			}
		}

		// Token: 0x06000149 RID: 329 RVA: 0x0000C358 File Offset: 0x0000A558
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

		// Token: 0x0600014A RID: 330 RVA: 0x0000C431 File Offset: 0x0000A631
		private void _tooltip_IsVisibleChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			if (this._tooltip.IsOpen)
			{
				this._tooltipTimer.Start();
			}
		}

		// Token: 0x0600014B RID: 331 RVA: 0x0000C44C File Offset: 0x0000A64C
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

		// Token: 0x0600014C RID: 332 RVA: 0x0000C4DF File Offset: 0x0000A6DF
		private void AssociatedObject_PreviewMouseDown(object sender, MouseButtonEventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x0600014D RID: 333 RVA: 0x0000C4E7 File Offset: 0x0000A6E7
		private void _timer_Tick(object sender, EventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x0600014E RID: 334 RVA: 0x0000C4EF File Offset: 0x0000A6EF
		private void AssociatedObject_PreviewKeyDown(object sender, KeyEventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x0600014F RID: 335 RVA: 0x0000C4F7 File Offset: 0x0000A6F7
		private void CloseTooltip()
		{
			this._tooltip.Content = null;
			this._tooltip.IsOpen = false;
			this._tooltipTimer.Stop();
		}

		// Token: 0x06000150 RID: 336 RVA: 0x0000C51C File Offset: 0x0000A71C
		private void AssociatedObject_MouseMove(object sender, MouseEventArgs e)
		{
			e.Handled = true;
			if (this.Key == null)
			{
				return;
			}
			if (!SettingManager.Get().GetTzpManger(this.Key).IsDiff)
			{
				return;
			}
			if (SettingManager.Get().GetTzpManger(this.Key).Type != TzpType.Code && SettingManager.Get().GetTzpManger(this.Key).Type != TzpType.Report)
			{
				return;
			}
			TextViewPosition? positionFromPoint = base.AssociatedObject.GetPositionFromPoint(e.GetPosition(base.AssociatedObject));
			if (positionFromPoint != null)
			{
				DocumentLine lineUnderMouse = base.AssociatedObject.Document.GetLineUnderMouse(positionFromPoint.Value);
				this._tooltip.Content = base.AssociatedObject.Document.GetText(lineUnderMouse.Offset, lineUnderMouse.EndOffset - lineUnderMouse.Offset);
				this._tooltip.IsOpen = true;
				this._tooltipTimer.Start();
			}
		}

		// Token: 0x06000151 RID: 337 RVA: 0x0000C606 File Offset: 0x0000A806
		private void AssociatedObject_MouseLeave(object sender, MouseEventArgs e)
		{
			this.CloseTooltip();
		}

		// Token: 0x0400009D RID: 157
		private ToolTip _tooltip;

		// Token: 0x0400009E RID: 158
		private DispatcherTimer _tooltipTimer;
	}
}
