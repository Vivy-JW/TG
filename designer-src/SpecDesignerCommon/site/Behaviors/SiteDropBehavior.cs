using System;
using System.Windows;
using System.Windows.Interactivity;

namespace SpecDesignerCommon.Site.Behaviors
{
	// Token: 0x02000036 RID: 54
	public class SiteDropBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x060001D0 RID: 464 RVA: 0x000086B0 File Offset: 0x000068B0
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.AllowDrop = true;
			base.AssociatedObject.DragEnter += this.AssociatedObject_DragEnter;
			base.AssociatedObject.DragLeave += this.AssociatedObject_DragLeave;
			base.AssociatedObject.DragOver += this.AssociatedObject_DragOver;
			base.AssociatedObject.Drop += this.AssociatedObject_Drop;
		}

		// Token: 0x060001D1 RID: 465 RVA: 0x0000872C File Offset: 0x0000692C
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

		// Token: 0x060001D2 RID: 466 RVA: 0x000087A4 File Offset: 0x000069A4
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
				IDragable dragable = e.Data.GetData(this.type) as IDragable;
				if (dropable.CanDrop(dragable))
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

		// Token: 0x060001D3 RID: 467 RVA: 0x0000885A File Offset: 0x00006A5A
		private void AssociatedObject_DragLeave(object sender, DragEventArgs e)
		{
			if (this.adorner != null)
			{
				this.adorner.Remove();
			}
			e.Handled = true;
		}

		// Token: 0x060001D4 RID: 468 RVA: 0x00008878 File Offset: 0x00006A78
		private void AssociatedObject_DragEnter(object sender, DragEventArgs e)
		{
			if (this.adorner == null)
			{
				this.adorner = new AllowDropAdorner(sender as UIElement);
			}
			if (base.AssociatedObject.AllowDrop && base.AssociatedObject.DataContext != null)
			{
				IDropable dropable = base.AssociatedObject.DataContext as IDropable;
				if (dropable != null)
				{
					this.type = dropable.AllowType;
					dropable.DropOver(e);
				}
			}
			e.Handled = true;
		}

		// Token: 0x060001D5 RID: 469 RVA: 0x000088E8 File Offset: 0x00006AE8
		protected override void OnDetaching()
		{
			base.OnDetaching();
			base.AssociatedObject.DragEnter -= this.AssociatedObject_DragEnter;
			base.AssociatedObject.DragLeave -= this.AssociatedObject_DragLeave;
			base.AssociatedObject.DragOver -= this.AssociatedObject_DragOver;
			base.AssociatedObject.Drop -= this.AssociatedObject_Drop;
		}

		// Token: 0x040000AA RID: 170
		private Type type;

		// Token: 0x040000AB RID: 171
		private AllowDropAdorner adorner;
	}
}
