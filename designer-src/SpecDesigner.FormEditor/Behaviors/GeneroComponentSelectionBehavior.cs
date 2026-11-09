using System;
using System.ComponentModel;
using System.Windows;
using System.Windows.Input;
using System.Windows.Interactivity;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x02000024 RID: 36
	public class GeneroComponentSelectionBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x0600012B RID: 299 RVA: 0x00006E64 File Offset: 0x00005064
		protected override void OnAttached()
		{
			base.OnAttached();
			if (base.AssociatedObject.DataContext == null || !(base.AssociatedObject.DataContext is XmlElement))
			{
				return;
			}
			this.source = base.AssociatedObject.DataContext as XmlElement;
			this.source.PropertyChanged += this.source_PropertyChanged;
		}

		// Token: 0x0600012C RID: 300 RVA: 0x00006EC4 File Offset: 0x000050C4
		private void source_PropertyChanged(object sender, PropertyChangedEventArgs e)
		{
			if (e.PropertyName == "IsSelected" && this.source.IsSelected)
			{
				base.AssociatedObject.BringIntoView();
				base.AssociatedObject.Focus();
				Keyboard.Focus(base.AssociatedObject);
			}
		}

		// Token: 0x0600012D RID: 301 RVA: 0x00006F13 File Offset: 0x00005113
		protected override void OnDetaching()
		{
			this.source.PropertyChanged -= this.source_PropertyChanged;
			base.OnDetaching();
		}

		// Token: 0x040000A5 RID: 165
		private XmlElement source;
	}
}
