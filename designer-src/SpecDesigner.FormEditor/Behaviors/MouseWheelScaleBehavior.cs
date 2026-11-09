using System;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Windows.Media;
using SpecDesigner.Controls.Controls;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x0200000B RID: 11
	public class MouseWheelScaleBehavior : Behavior<Grid>
	{
		// Token: 0x0600002C RID: 44 RVA: 0x00002BD0 File Offset: 0x00000DD0
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.MouseWheel += this.AssociatedObject_MouseWheel;
			if (!(base.AssociatedObject.RenderTransform is ScaleTransform))
			{
				base.AssociatedObject.RenderTransform = new ScaleTransform(1.0, 1.0, 0.0, 0.0);
			}
		}

		// Token: 0x0600002D RID: 45 RVA: 0x00002C44 File Offset: 0x00000E44
		private void AssociatedObject_MouseWheel(object sender, MouseWheelEventArgs e)
		{
			if (Keyboard.Modifiers == ModifierKeys.Control)
			{
				Transform renderTransform = base.AssociatedObject.RenderTransform;
				if (e.Delta < 0)
				{
					ScrollBarCommands.ZoomInCommand.Execute(null, base.AssociatedObject);
				}
				else
				{
					ScrollBarCommands.ZoomOutCommand.Execute(null, base.AssociatedObject);
				}
				e.Handled = true;
			}
		}

		// Token: 0x0600002E RID: 46 RVA: 0x00002C99 File Offset: 0x00000E99
		protected override void OnDetaching()
		{
			base.AssociatedObject.MouseWheel -= this.AssociatedObject_MouseWheel;
		}
	}
}
