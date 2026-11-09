using System;
using System.Windows;
using System.Windows.Input;
using System.Windows.Interactivity;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x02000057 RID: 87
	public class KeyBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x0600036E RID: 878 RVA: 0x00012E7D File Offset: 0x0001107D
		public KeyBehavior()
		{
		}

		// Token: 0x0600036F RID: 879 RVA: 0x00012E85 File Offset: 0x00011085
		public KeyBehavior(PackageKey key)
		{
			this._key = key;
		}

		// Token: 0x06000370 RID: 880 RVA: 0x00012E94 File Offset: 0x00011094
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.AddHandler(UIElement.KeyUpEvent, new KeyEventHandler(this.KeyUp));
			base.AssociatedObject.AddHandler(UIElement.PreviewKeyDownEvent, new KeyEventHandler(this.PreviewKeyDown), true);
		}

		// Token: 0x06000371 RID: 881 RVA: 0x00012EE0 File Offset: 0x000110E0
		protected override void OnDetaching()
		{
			base.AssociatedObject.RemoveHandler(UIElement.KeyUpEvent, new KeyEventHandler(this.KeyUp));
			base.AssociatedObject.RemoveHandler(UIElement.PreviewKeyDownEvent, new KeyEventHandler(this.PreviewKeyDown));
			base.OnDetaching();
		}

		// Token: 0x06000372 RID: 882 RVA: 0x00012F20 File Offset: 0x00011120
		protected void PreviewKeyDown(object sender, KeyEventArgs args)
		{
			Key key = args.Key;
			switch (key)
			{
			case Key.Left:
			case Key.Up:
			case Key.Right:
			case Key.Down:
				break;
			default:
				if (key != Key.Delete)
				{
					return;
				}
				break;
			}
			args.Handled = true;
		}

		// Token: 0x06000373 RID: 883 RVA: 0x00012F5C File Offset: 0x0001115C
		protected void KeyUp(object sender, KeyEventArgs args)
		{
			Key key = args.Key;
			switch (key)
			{
			case Key.Left:
				this.SelectedComponentsMoveLeft();
				args.Handled = true;
				return;
			case Key.Up:
				this.SelectedComponentsMoveUp();
				args.Handled = true;
				return;
			case Key.Right:
				this.SelectedComponentsMoveRight();
				args.Handled = true;
				return;
			case Key.Down:
				this.SelectedComponentsMoveDown();
				args.Handled = true;
				return;
			default:
				if (key != Key.Delete)
				{
					return;
				}
				ComponentHelper.Get(this._key).DeleteSelection();
				args.Handled = true;
				return;
			}
		}

		// Token: 0x06000374 RID: 884 RVA: 0x00012FDE File Offset: 0x000111DE
		private void SelectedComponentsMoveRight()
		{
			if (null == this._key)
			{
				return;
			}
			ComponentHelper.Get(this._key).Move(MoveDirection.Right, 1);
		}

		// Token: 0x06000375 RID: 885 RVA: 0x00013001 File Offset: 0x00011201
		private void SelectedComponentsMoveLeft()
		{
			if (null == this._key)
			{
				return;
			}
			ComponentHelper.Get(this._key).Move(MoveDirection.Left, 1);
		}

		// Token: 0x06000376 RID: 886 RVA: 0x00013024 File Offset: 0x00011224
		private void SelectedComponentsMoveDown()
		{
			if (null == this._key)
			{
				return;
			}
			ComponentHelper.Get(this._key).Move(MoveDirection.Down, 1);
		}

		// Token: 0x06000377 RID: 887 RVA: 0x00013047 File Offset: 0x00011247
		private void SelectedComponentsMoveUp()
		{
			if (null == this._key)
			{
				return;
			}
			ComponentHelper.Get(this._key).Move(MoveDirection.Up, 1);
		}

		// Token: 0x040001C1 RID: 449
		private PackageKey _key;
	}
}
