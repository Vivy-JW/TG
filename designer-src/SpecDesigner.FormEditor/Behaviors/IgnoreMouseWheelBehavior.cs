using System;
using System.Windows;
using System.Windows.Input;
using System.Windows.Interactivity;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x0200004D RID: 77
	public sealed class IgnoreMouseWheelBehavior : Behavior<UIElement>
	{
		// Token: 0x060002CB RID: 715 RVA: 0x0000E1AC File Offset: 0x0000C3AC
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.PreviewMouseWheel += this.AssociatedObject_PreviewMouseWheel;
		}

		// Token: 0x060002CC RID: 716 RVA: 0x0000E1CB File Offset: 0x0000C3CB
		protected override void OnDetaching()
		{
			base.AssociatedObject.PreviewMouseWheel -= this.AssociatedObject_PreviewMouseWheel;
			base.OnDetaching();
		}

		// Token: 0x060002CD RID: 717 RVA: 0x0000E1EC File Offset: 0x0000C3EC
		private void AssociatedObject_PreviewMouseWheel(object sender, MouseWheelEventArgs e)
		{
			e.Handled = true;
			MouseWheelEventArgs e2 = new MouseWheelEventArgs(e.MouseDevice, e.Timestamp, e.Delta);
			e2.RoutedEvent = UIElement.MouseWheelEvent;
			base.AssociatedObject.RaiseEvent(e2);
		}
	}
}
