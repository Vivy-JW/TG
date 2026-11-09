using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interactivity;

namespace SpecDesignerCommon.Connection.Adorner
{
	// Token: 0x0200011A RID: 282
	public class PasswordBoxWatermarkBehavior : Behavior<PasswordBox>
	{
		// Token: 0x06000A04 RID: 2564 RVA: 0x00031FD8 File Offset: 0x000301D8
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.Loaded += this.AssociatedObjectLoaded;
			base.AssociatedObject.GotKeyboardFocus += this.AssociatedObject_GotKeyboardFocus;
			base.AssociatedObject.LostKeyboardFocus += this.AssociatedObject_LostKeyboardFocus;
		}

		// Token: 0x06000A05 RID: 2565 RVA: 0x00032030 File Offset: 0x00030230
		protected override void OnDetaching()
		{
			base.AssociatedObject.Loaded -= this.AssociatedObjectLoaded;
			base.AssociatedObject.GotKeyboardFocus -= this.AssociatedObject_GotKeyboardFocus;
			base.AssociatedObject.LostKeyboardFocus -= this.AssociatedObject_LostKeyboardFocus;
			base.OnDetaching();
		}

		// Token: 0x06000A06 RID: 2566 RVA: 0x00032088 File Offset: 0x00030288
		private void AssociatedObject_GotKeyboardFocus(object sender, KeyboardFocusChangedEventArgs e)
		{
			this.RemoveAdorner();
		}

		// Token: 0x06000A07 RID: 2567 RVA: 0x00032090 File Offset: 0x00030290
		private void AssociatedObjectLoaded(object sender, RoutedEventArgs e)
		{
			this.AddAdorner();
		}

		// Token: 0x06000A08 RID: 2568 RVA: 0x00032098 File Offset: 0x00030298
		private void AssociatedObject_LostKeyboardFocus(object sender, KeyboardFocusChangedEventArgs e)
		{
			this.AddAdorner();
		}

		// Token: 0x06000A09 RID: 2569 RVA: 0x000320A0 File Offset: 0x000302A0
		private void AddAdorner()
		{
			if (this.adorner == null)
			{
				this.adorner = new WaterMarkeAdorner(base.AssociatedObject, this.SECRETWORD);
			}
			base.AssociatedObject.TryAddAdorner<WaterMarkeAdorner>(this.adorner);
		}

		// Token: 0x06000A0A RID: 2570 RVA: 0x000320D2 File Offset: 0x000302D2
		private void RemoveAdorner()
		{
			base.AssociatedObject.TryRemoveAdorners<WaterMarkeAdorner>();
		}

		// Token: 0x040003C7 RID: 967
		private readonly string SECRETWORD = "**********";

		// Token: 0x040003C8 RID: 968
		private WaterMarkeAdorner adorner;
	}
}
