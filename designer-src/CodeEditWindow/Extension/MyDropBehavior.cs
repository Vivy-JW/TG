using System;
using System.Windows;
using System.Windows.Interactivity;
using SpecDesigner.Infrastructure.Extension;

namespace SpecDesigner.CodeEditWindow.Extension
{
	// Token: 0x0200001D RID: 29
	public class MyDropBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x0600010E RID: 270 RVA: 0x0000AE78 File Offset: 0x00009078
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.AllowDrop = true;
			base.AssociatedObject.DragEnter += this.AssociatedObject_DragEnter;
			base.AssociatedObject.DragLeave += this.AssociatedObject_DragLeave;
			base.AssociatedObject.DragOver += this.AssociatedObject_DragOver;
			base.AssociatedObject.Drop += this.AssociatedObject_Drop;
		}

		// Token: 0x0600010F RID: 271 RVA: 0x0000AEF4 File Offset: 0x000090F4
		private void AssociatedObject_Drop(object sender, DragEventArgs e)
		{
			if (this.type != null && e.Data.GetDataPresent(this.type))
			{
				IDropable dropable = base.AssociatedObject.DataContext as IDropable;
				dropable.Drop(e.Data.GetData(this.type) as IDragable);
			}
			if (this.adorner != null)
			{
				this.adorner.Remove();
			}
			e.Handled = true;
		}

		// Token: 0x06000110 RID: 272 RVA: 0x0000AF6C File Offset: 0x0000916C
		private void AssociatedObject_DragOver(object sender, DragEventArgs e)
		{
			if (this.adorner != null)
			{
				this.adorner.Remove();
			}
			IDropable dropable = base.AssociatedObject.DataContext as IDropable;
			e.Effects = DragDropEffects.None;
			if (this.type != null && e.Data.GetDataPresent(this.type))
			{
				e.Data.GetData(this.type);
				if (dropable.CanDrop)
				{
					e.Effects = DragDropEffects.Move;
					if (this.adorner != null)
					{
						this.adorner.Update();
					}
				}
				else
				{
					e.Effects = DragDropEffects.None;
					if (this.adorner != null)
					{
						this.adorner.Remove();
					}
				}
			}
			e.Handled = true;
		}

		// Token: 0x06000111 RID: 273 RVA: 0x0000B01C File Offset: 0x0000921C
		private void AssociatedObject_DragLeave(object sender, DragEventArgs e)
		{
			if (this.adorner != null)
			{
				this.adorner.Remove();
			}
			e.Handled = true;
		}

		// Token: 0x06000112 RID: 274 RVA: 0x0000B038 File Offset: 0x00009238
		private void AssociatedObject_DragEnter(object sender, DragEventArgs e)
		{
			if (base.AssociatedObject.AllowDrop && base.AssociatedObject.DataContext != null)
			{
				IDropable dropable = base.AssociatedObject.DataContext as IDropable;
				if (dropable != null && dropable.CanDrop)
				{
					if (this.adorner == null)
					{
						this.adorner = new AllowDropAdorner(sender as UIElement);
					}
					this.type = dropable.AllowType;
					dropable.DropOver(e);
				}
			}
			e.Handled = true;
		}

		// Token: 0x06000113 RID: 275 RVA: 0x0000B0B0 File Offset: 0x000092B0
		protected override void OnDetaching()
		{
			base.OnDetaching();
			base.AssociatedObject.DragEnter -= this.AssociatedObject_DragEnter;
			base.AssociatedObject.DragLeave -= this.AssociatedObject_DragLeave;
			base.AssociatedObject.DragOver -= this.AssociatedObject_DragOver;
			base.AssociatedObject.Drop -= this.AssociatedObject_Drop;
		}

		// Token: 0x04000085 RID: 133
		private Type type;

		// Token: 0x04000086 RID: 134
		private AllowDropAdorner adorner;
	}
}
