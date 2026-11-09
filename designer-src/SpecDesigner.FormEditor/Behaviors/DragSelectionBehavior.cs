using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Windows.Media;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x02000055 RID: 85
	public class DragSelectionBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x06000363 RID: 867 RVA: 0x0001292C File Offset: 0x00010B2C
		protected override void OnAttached()
		{
			base.OnAttached();
			SpecificationInfo specificationInfo = base.AssociatedObject.DataContext as SpecificationInfo;
			if (specificationInfo == null)
			{
				return;
			}
			this._key = specificationInfo.Key;
			base.AssociatedObject.PreviewMouseLeftButtonDown += this.AssociatedObject_PreviewMouseLeftButtonDown;
			base.AssociatedObject.PreviewMouseLeftButtonUp += this.AssociatedObject_PreviewMouseLeftButtonUp;
			base.AssociatedObject.PreviewMouseMove += this.AssociatedObject_PreviewMouseMove;
		}

		// Token: 0x06000364 RID: 868 RVA: 0x000129A8 File Offset: 0x00010BA8
		protected override void OnDetaching()
		{
			base.AssociatedObject.PreviewMouseLeftButtonDown -= this.AssociatedObject_PreviewMouseLeftButtonDown;
			base.AssociatedObject.PreviewMouseLeftButtonUp -= this.AssociatedObject_PreviewMouseLeftButtonUp;
			base.AssociatedObject.PreviewMouseMove -= this.AssociatedObject_PreviewMouseMove;
			base.OnDetaching();
		}

		// Token: 0x06000365 RID: 869 RVA: 0x00012A00 File Offset: 0x00010C00
		private void AssociatedObject_PreviewMouseMove(object sender, MouseEventArgs e)
		{
			if (this._isDragging && (Keyboard.IsKeyDown(Key.LeftShift) || Keyboard.IsKeyDown(Key.RightShift)))
			{
				this._rect.UpdateRectPoint(e.GetPosition(base.AssociatedObject));
			}
		}

		// Token: 0x06000366 RID: 870 RVA: 0x00012A34 File Offset: 0x00010C34
		private void AssociatedObject_PreviewMouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			base.AssociatedObject.ReleaseMouseCapture();
			if (this._isDragging)
			{
				this._layer.Remove(this._rect);
				this._endPoint = e.GetPosition(base.AssociatedObject);
				List<XmlElement> list = new List<XmlElement>();
				this.EnumChildrenWidget(base.AssociatedObject, ref list);
				ComponentHelper.Get(this._key).MultipleSelection(list);
			}
			this._isDragging = false;
		}

		// Token: 0x06000367 RID: 871 RVA: 0x00012AA4 File Offset: 0x00010CA4
		private void AssociatedObject_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
		{
			if (Keyboard.IsKeyDown(Key.LeftShift) || Keyboard.IsKeyDown(Key.RightShift))
			{
				ComponentHelper.Get(this._key).ClearSelection();
				if (this._layer == null)
				{
					this._layer = AdornerLayer.GetAdornerLayer(base.AssociatedObject);
				}
				if (this._rect == null)
				{
					this._rect = new DragSelectionRectAdorner(base.AssociatedObject);
				}
				this._rect.StartPoint = e.GetPosition(base.AssociatedObject);
				this._layer.Add(this._rect);
				this._startPoint = this._rect.StartPoint;
				this._isDragging = true;
				base.AssociatedObject.CaptureMouse();
				e.Handled = true;
			}
		}

		// Token: 0x06000368 RID: 872 RVA: 0x00012B5C File Offset: 0x00010D5C
		private void EnumChildrenWidget(Visual parent, ref List<XmlElement> collection)
		{
			for (int i = VisualTreeHelper.GetChildrenCount(parent) - 1; i >= 0; i--)
			{
				Visual visual = (Visual)VisualTreeHelper.GetChild(parent, i);
				FrameworkElement frameworkElement = visual as FrameworkElement;
				if (frameworkElement.IsVisible)
				{
					this.EnumChildrenWidget(visual, ref collection);
					if (frameworkElement != null && frameworkElement.DataContext is XmlElement)
					{
						MatrixTransform matrixTransform = (MatrixTransform)visual.TransformToVisual(base.AssociatedObject);
						if (DragSelectionBehavior.ContainBounds(this._startPoint, this._endPoint, new Point(matrixTransform.Matrix.OffsetX, matrixTransform.Matrix.OffsetY), frameworkElement.Width, frameworkElement.Height))
						{
							XmlElement xmlElement = frameworkElement.DataContext as XmlElement;
							if (!collection.Contains(xmlElement) && (collection.Count <= 0 || xmlElement.Parent == collection.FirstOrDefault<XmlElement>().Parent))
							{
								collection.Add(xmlElement);
							}
						}
					}
				}
			}
		}

		// Token: 0x06000369 RID: 873 RVA: 0x00012C58 File Offset: 0x00010E58
		public static bool ContainBounds(Point boundStart, Point boundEnd, Point p, double width, double height)
		{
			double num = Math.Abs(boundStart.X - boundEnd.X);
			double num2 = Math.Abs(boundStart.Y - boundEnd.Y);
			double num3 = Math.Min(boundStart.X, boundEnd.X);
			double num4 = Math.Min(boundStart.Y, boundEnd.Y);
			return num >= 0.0 && width >= 0.0 && (p.X <= num3 + num && p.X + width >= num3 && p.Y <= num4 + num2) && p.Y + height >= num4;
		}

		// Token: 0x040001B6 RID: 438
		private PackageKey _key;

		// Token: 0x040001B7 RID: 439
		private AdornerLayer _layer;

		// Token: 0x040001B8 RID: 440
		private DragSelectionRectAdorner _rect;

		// Token: 0x040001B9 RID: 441
		private TranslateTransform _transform = new TranslateTransform();

		// Token: 0x040001BA RID: 442
		private Point _startPoint;

		// Token: 0x040001BB RID: 443
		private Point _endPoint;

		// Token: 0x040001BC RID: 444
		private bool _isDragging;
	}
}
