using System;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Interactivity;
using SpecDesigner.FormEditor.Views;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x0200002C RID: 44
	public class RepeatAdornerBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x0600018D RID: 397 RVA: 0x0000878A File Offset: 0x0000698A
		protected override void OnAttached()
		{
			base.OnAttached();
			this.CreateRepeatAdorner();
		}

		// Token: 0x0600018E RID: 398 RVA: 0x00008798 File Offset: 0x00006998
		protected override void OnDetaching()
		{
			if (this._adornerLayer != null && this._adorner != null)
			{
				this._adornerLayer.Remove(this._adorner);
				this._adorner.Dispose();
			}
			base.OnDetaching();
		}

		// Token: 0x0600018F RID: 399 RVA: 0x000087CC File Offset: 0x000069CC
		private void CreateRepeatAdorner()
		{
			this._adornerLayer = AdornerLayer.GetAdornerLayer(base.AssociatedObject);
			if (this._adornerLayer == null)
			{
				return;
			}
			this._adorner = new WidgetRepeatAdorner(base.AssociatedObject);
			this._adornerLayer.Add(this._adorner);
		}

		// Token: 0x040000E3 RID: 227
		private AdornerLayer _adornerLayer;

		// Token: 0x040000E4 RID: 228
		private WidgetRepeatAdorner _adorner;
	}
}
