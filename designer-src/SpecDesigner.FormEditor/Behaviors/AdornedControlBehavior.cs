using System;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Interactivity;
using SpecDesigner.Controls.AdornedControl;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x0200001B RID: 27
	public class AdornedControlBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x1700002E RID: 46
		// (get) Token: 0x060000D9 RID: 217 RVA: 0x00005372 File Offset: 0x00003572
		// (set) Token: 0x060000DA RID: 218 RVA: 0x00005384 File Offset: 0x00003584
		public FrameworkElement AdornerContent
		{
			get
			{
				return (FrameworkElement)base.GetValue(AdornedControlBehavior.AdornerContentProperty);
			}
			set
			{
				base.SetValue(AdornedControlBehavior.AdornerContentProperty, value);
			}
		}

		// Token: 0x060000DB RID: 219 RVA: 0x00005394 File Offset: 0x00003594
		public static void IsAdornerVisible_PropertyChanged(DependencyObject sender, DependencyPropertyChangedEventArgs e)
		{
			AdornedControlBehavior adornedControlBehavior = (AdornedControlBehavior)sender;
			adornedControlBehavior.ShowOrHideAdornerInternal();
		}

		// Token: 0x1700002F RID: 47
		// (get) Token: 0x060000DC RID: 220 RVA: 0x000053AE File Offset: 0x000035AE
		// (set) Token: 0x060000DD RID: 221 RVA: 0x000053C0 File Offset: 0x000035C0
		public bool IsAdornerVisible
		{
			get
			{
				return (bool)base.GetValue(AdornedControlBehavior.IsAdornerVisibleProperty);
			}
			set
			{
				base.SetValue(AdornedControlBehavior.IsAdornerVisibleProperty, value);
			}
		}

		// Token: 0x060000DE RID: 222 RVA: 0x000053D3 File Offset: 0x000035D3
		private void ShowOrHideAdornerInternal()
		{
			this.AdornerContent.Visibility = (this.IsAdornerVisible ? Visibility.Visible : Visibility.Collapsed);
		}

		// Token: 0x17000030 RID: 48
		// (get) Token: 0x060000DF RID: 223 RVA: 0x000053EC File Offset: 0x000035EC
		// (set) Token: 0x060000E0 RID: 224 RVA: 0x000053FE File Offset: 0x000035FE
		public XmlElement BindingSource
		{
			get
			{
				return (XmlElement)base.GetValue(AdornedControlBehavior.BindingSourceProperty);
			}
			set
			{
				base.SetValue(AdornedControlBehavior.BindingSourceProperty, value);
			}
		}

		// Token: 0x060000E1 RID: 225 RVA: 0x0000540C File Offset: 0x0000360C
		protected override void OnAttached()
		{
			base.OnAttached();
			base.AssociatedObject.Loaded += this.AssociatedObject_Loaded;
		}

		// Token: 0x060000E2 RID: 226 RVA: 0x0000542B File Offset: 0x0000362B
		private void AssociatedObject_Loaded(object sender, RoutedEventArgs e)
		{
			base.AssociatedObject.Loaded -= this.AssociatedObject_Loaded;
			this.CreateAdorner();
		}

		// Token: 0x060000E3 RID: 227 RVA: 0x0000544A File Offset: 0x0000364A
		protected override void OnDetaching()
		{
			this.DisposeAdorner();
			base.OnDetaching();
		}

		// Token: 0x060000E4 RID: 228 RVA: 0x00005458 File Offset: 0x00003658
		internal void CreateAdorner()
		{
			if (this.AdornerContent == null)
			{
				return;
			}
			this.AdornerContent.DataContext = this.BindingSource;
			if (this._adornerLayer == null)
			{
				this._adornerLayer = AdornerLayer.GetAdornerLayer(base.AssociatedObject);
			}
			if (this._adornerLayer == null)
			{
				return;
			}
			if (this._adorner == null)
			{
				this._adorner = new FrameworkElementAdorner(this.AdornerContent, base.AssociatedObject);
				this._adornerLayer.Add(this._adorner);
			}
			this.ShowOrHideAdornerInternal();
		}

		// Token: 0x060000E5 RID: 229 RVA: 0x000054D8 File Offset: 0x000036D8
		private void DisposeAdorner()
		{
			if (this._adornerLayer != null && this._adorner != null)
			{
				this._adornerLayer.Remove(this._adorner);
				this._adorner.Dispose();
				if (this.AdornerContent is IDisposable)
				{
					(this.AdornerContent as IDisposable).Dispose();
				}
			}
		}

		// Token: 0x04000081 RID: 129
		public static readonly DependencyProperty AdornerContentProperty = DependencyProperty.Register("AdornerContent", typeof(FrameworkElement), typeof(AdornedControlBehavior));

		// Token: 0x04000082 RID: 130
		public static readonly DependencyProperty IsAdornerVisibleProperty = DependencyProperty.Register("IsAdornerVisible", typeof(bool), typeof(AdornedControlBehavior), new FrameworkPropertyMetadata(false, new PropertyChangedCallback(AdornedControlBehavior.IsAdornerVisible_PropertyChanged)));

		// Token: 0x04000083 RID: 131
		public static readonly DependencyProperty BindingSourceProperty = DependencyProperty.Register("BindingSource", typeof(XmlElement), typeof(AdornedControlBehavior));

		// Token: 0x04000084 RID: 132
		private AdornerLayer _adornerLayer;

		// Token: 0x04000085 RID: 133
		private FrameworkElementAdorner _adorner;
	}
}
