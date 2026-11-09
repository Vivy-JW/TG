using System;
using System.Windows;
using System.Windows.Input;
using System.Windows.Interactivity;

namespace SpecDesignerCommon.Site.Behaviors
{
	// Token: 0x02000037 RID: 55
	public class SiteDragBehavrior : Behavior<FrameworkElement>
	{
		// Token: 0x060001D7 RID: 471 RVA: 0x00008960 File Offset: 0x00006B60
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.MouseLeftButtonDown += this.AssociatedObject_MouseLeftButtonDown;
			base.AssociatedObject.MouseLeftButtonUp += this.AssociatedObject_MouseLeftButtonUp;
			base.AssociatedObject.MouseLeave += this.AssociatedObject_MouseLeave;
		}

		// Token: 0x060001D8 RID: 472 RVA: 0x000089B8 File Offset: 0x00006BB8
		private void AssociatedObject_MouseLeave(object sender, MouseEventArgs e)
		{
			if (this.isMouseClicked && base.AssociatedObject.DataContext != null)
			{
				IDragable dragable = base.AssociatedObject.DataContext as IDragable;
				if (dragable != null)
				{
					DataObject dataObject = new DataObject();
					dataObject.SetData(dragable.DragType, base.AssociatedObject.DataContext);
					DragDrop.DoDragDrop(base.AssociatedObject, dataObject, DragDropEffects.Move);
				}
			}
			this.isMouseClicked = false;
		}

		// Token: 0x060001D9 RID: 473 RVA: 0x00008A20 File Offset: 0x00006C20
		private void AssociatedObject_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			this.isMouseClicked = false;
		}

		// Token: 0x060001DA RID: 474 RVA: 0x00008A29 File Offset: 0x00006C29
		private void AssociatedObject_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			this.isMouseClicked = true;
		}

		// Token: 0x060001DB RID: 475 RVA: 0x00008A34 File Offset: 0x00006C34
		protected override void OnDetaching()
		{
			base.OnDetaching();
			base.AssociatedObject.MouseLeftButtonDown -= this.AssociatedObject_MouseLeftButtonDown;
			base.AssociatedObject.MouseLeftButtonUp -= this.AssociatedObject_MouseLeftButtonUp;
			base.AssociatedObject.MouseLeave -= this.AssociatedObject_MouseLeave;
		}

		// Token: 0x040000AC RID: 172
		private bool isMouseClicked;
	}
}
