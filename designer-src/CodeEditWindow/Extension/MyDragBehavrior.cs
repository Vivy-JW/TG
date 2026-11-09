using System;
using System.Windows;
using System.Windows.Input;
using System.Windows.Interactivity;
using SpecDesigner.Infrastructure.Extension;

namespace SpecDesigner.CodeEditWindow.Extension
{
	// Token: 0x0200001E RID: 30
	public class MyDragBehavrior : Behavior<FrameworkElement>
	{
		// Token: 0x06000115 RID: 277 RVA: 0x0000B128 File Offset: 0x00009328
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.MouseLeftButtonDown += this.AssociatedObject_MouseLeftButtonDown;
			base.AssociatedObject.MouseLeftButtonUp += this.AssociatedObject_MouseLeftButtonUp;
			base.AssociatedObject.MouseLeave += this.AssociatedObject_MouseLeave;
		}

		// Token: 0x06000116 RID: 278 RVA: 0x0000B180 File Offset: 0x00009380
		private void AssociatedObject_MouseLeave(object sender, MouseEventArgs e)
		{
			if (this.isMouseClicked)
			{
				if (base.AssociatedObject.DataContext != null)
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
		}

		// Token: 0x06000117 RID: 279 RVA: 0x0000B1E8 File Offset: 0x000093E8
		private void AssociatedObject_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			this.isMouseClicked = false;
		}

		// Token: 0x06000118 RID: 280 RVA: 0x0000B1F4 File Offset: 0x000093F4
		private void AssociatedObject_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			IDragable dragable = base.AssociatedObject.DataContext as IDragable;
			if (dragable != null && dragable.CanDrag)
			{
				this.isMouseClicked = true;
			}
		}

		// Token: 0x06000119 RID: 281 RVA: 0x0000B224 File Offset: 0x00009424
		protected override void OnDetaching()
		{
			base.OnDetaching();
			base.AssociatedObject.MouseLeftButtonDown -= this.AssociatedObject_MouseLeftButtonDown;
			base.AssociatedObject.MouseLeftButtonUp -= this.AssociatedObject_MouseLeftButtonUp;
			base.AssociatedObject.MouseLeave -= this.AssociatedObject_MouseLeave;
		}

		// Token: 0x04000087 RID: 135
		private bool isMouseClicked;
	}
}
