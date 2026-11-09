using System;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Media;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x0200000D RID: 13
	public class VisualComponentsAdorner : Adorner
	{
		// Token: 0x06000039 RID: 57 RVA: 0x000034DA File Offset: 0x000016DA
		public VisualComponentsAdorner(UIElement element, UIElement elementToShow)
			: base(element)
		{
			this.elementToShow = elementToShow;
		}

		// Token: 0x0600003A RID: 58 RVA: 0x000034EA File Offset: 0x000016EA
		protected override Size MeasureOverride(Size constraint)
		{
			this.elementToShow.Measure(constraint);
			return constraint;
		}

		// Token: 0x0600003B RID: 59 RVA: 0x000034F9 File Offset: 0x000016F9
		protected override Size ArrangeOverride(Size finalSize)
		{
			this.elementToShow.Arrange(new Rect(finalSize));
			return finalSize;
		}

		// Token: 0x0600003C RID: 60 RVA: 0x0000350D File Offset: 0x0000170D
		protected override Visual GetVisualChild(int index)
		{
			return this.elementToShow;
		}

		// Token: 0x17000002 RID: 2
		// (get) Token: 0x0600003D RID: 61 RVA: 0x00003515 File Offset: 0x00001715
		protected override int VisualChildrenCount
		{
			get
			{
				return 1;
			}
		}

		// Token: 0x0600003E RID: 62 RVA: 0x00003518 File Offset: 0x00001718
		public override GeneralTransform GetDesiredTransform(GeneralTransform transform)
		{
			return new GeneralTransformGroup
			{
				Children = 
				{
					transform,
					new TranslateTransform(this.position.X, this.position.Y)
				}
			};
		}

		// Token: 0x0600003F RID: 63 RVA: 0x00003560 File Offset: 0x00001760
		public void UpdatePosition(Point point)
		{
			this.position = point;
			AdornerLayer adornerLayer = base.Parent as AdornerLayer;
			if (adornerLayer != null)
			{
				adornerLayer.Update(base.AdornedElement);
			}
		}

		// Token: 0x0400002D RID: 45
		private UIElement elementToShow;

		// Token: 0x0400002E RID: 46
		private Point position;
	}
}
