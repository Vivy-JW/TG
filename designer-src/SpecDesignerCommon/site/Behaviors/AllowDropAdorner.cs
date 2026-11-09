using System;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Media;
using SpecDesignerCommon.Site.Views;

namespace SpecDesignerCommon.Site.Behaviors
{
	// Token: 0x02000038 RID: 56
	internal class AllowDropAdorner : Adorner
	{
		// Token: 0x060001DD RID: 477 RVA: 0x00008A94 File Offset: 0x00006C94
		public AllowDropAdorner(UIElement adornedElement)
			: base(adornedElement)
		{
			base.Visibility = Visibility.Collapsed;
			this.adornerLayer = AdornerLayer.GetAdornerLayer(adornedElement);
			this.visualCollection = new VisualCollection(this);
			this.visualCollection.Add(this.uc);
			this.uc.Width = 16.0;
			this.uc.Height = 16.0;
			this.adornerLayer.Add(this);
		}

		// Token: 0x1700006F RID: 111
		// (get) Token: 0x060001DE RID: 478 RVA: 0x00008B18 File Offset: 0x00006D18
		protected override int VisualChildrenCount
		{
			get
			{
				return this.visualCollection.Count;
			}
		}

		// Token: 0x060001DF RID: 479 RVA: 0x00008B25 File Offset: 0x00006D25
		protected override Visual GetVisualChild(int index)
		{
			return this.visualCollection[index];
		}

		// Token: 0x060001E0 RID: 480 RVA: 0x00008B33 File Offset: 0x00006D33
		internal void Update()
		{
			this.adornerLayer.Update(base.AdornedElement);
			base.Visibility = Visibility.Visible;
		}

		// Token: 0x060001E1 RID: 481 RVA: 0x00008B4D File Offset: 0x00006D4D
		public void Remove()
		{
			base.Visibility = Visibility.Collapsed;
		}

		// Token: 0x060001E2 RID: 482 RVA: 0x00008B58 File Offset: 0x00006D58
		protected override Size ArrangeOverride(Size finalSize)
		{
			double width = base.AdornedElement.DesiredSize.Width;
			double height = base.AdornedElement.DesiredSize.Height;
			this.uc.Arrange(new Rect(width, height - this.uc.Height, this.uc.Width, this.uc.Height));
			return finalSize;
		}

		// Token: 0x040000AD RID: 173
		private VisualCollection visualCollection;

		// Token: 0x040000AE RID: 174
		private BlockArrow uc = new BlockArrow();

		// Token: 0x040000AF RID: 175
		private AdornerLayer adornerLayer;
	}
}
