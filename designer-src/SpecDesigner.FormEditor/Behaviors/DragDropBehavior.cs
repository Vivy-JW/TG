using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Documents;
using System.Windows.Input;
using System.Windows.Interactivity;
using SpecDesigner.Controls.AdornedControl;
using SpecDesigner.Controls.Controls;
using SpecDesigner.FormEditor.Helpers;
using SpecDesigner.FormEditor.Views;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Behaviors
{
	// Token: 0x02000034 RID: 52
	public class DragDropBehavior : Behavior<FrameworkElement>
	{
		// Token: 0x060001F1 RID: 497 RVA: 0x0000A1EC File Offset: 0x000083EC
		public DragDropBehavior(PackageKey key)
		{
			this._key = key;
			this._selectedComponents = new List<XmlElement>();
			Uri uri = new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Add.cur", UriKind.Absolute);
			Uri uri2 = new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Stop.cur", UriKind.Absolute);
			this._addCursor = new Cursor(Application.GetResourceStream(uri).Stream);
			this._denyCursor = new Cursor(Application.GetResourceStream(uri2).Stream);
		}

		// Token: 0x060001F2 RID: 498 RVA: 0x0000A274 File Offset: 0x00008474
		protected override void OnAttached()
		{
			base.OnAttached();
			this._container = base.AssociatedObject;
			base.AssociatedObject.AddHandler(UIElement.MouseDownEvent, new RoutedEventHandler(this.AssociatedObject_MouseDown), true);
			base.AssociatedObject.AddHandler(UIElement.MouseUpEvent, new MouseButtonEventHandler(this.AssociatedObject_MouseLeftButtonUp), true);
			base.AssociatedObject.AddHandler(UIElement.PreviewKeyUpEvent, new KeyEventHandler(this.Associated_PreviewKeyUp), true);
		}

		// Token: 0x060001F3 RID: 499 RVA: 0x0000A2EC File Offset: 0x000084EC
		protected override void OnDetaching()
		{
			base.AssociatedObject.RemoveHandler(UIElement.MouseDownEvent, new RoutedEventHandler(this.AssociatedObject_MouseDown));
			base.AssociatedObject.RemoveHandler(UIElement.MouseUpEvent, new MouseButtonEventHandler(this.AssociatedObject_MouseLeftButtonUp));
			base.AssociatedObject.RemoveHandler(UIElement.PreviewKeyUpEvent, new KeyEventHandler(this.Associated_PreviewKeyUp));
			this.ClearAdorner();
			base.OnDetaching();
		}

		// Token: 0x060001F4 RID: 500 RVA: 0x0000A359 File Offset: 0x00008559
		private void AssociatedObject_MouseUp(object sender, RoutedEventArgs e)
		{
			this.ClearAdorner();
		}

		// Token: 0x060001F5 RID: 501 RVA: 0x0000A364 File Offset: 0x00008564
		private void AssociatedObject_MouseDown(object sender, RoutedEventArgs e)
		{
			if (ComponentHelper.Get(this._key).IsSizeChanging)
			{
				return;
			}
			Point position = (e as MouseEventArgs).GetPosition(this._container);
			if (VisualTreeHelperEx.GetParentByPoint<AdornedControl>(this._container, position) == null)
			{
				return;
			}
			ManagedForm managedForm = sender as ManagedForm;
			if (managedForm != null && managedForm.CantMovableInSimpleForm())
			{
				return;
			}
			this._MouseDownStart = position;
			base.AssociatedObject.AddHandler(UIElement.MouseMoveEvent, new MouseEventHandler(this.AssociatedObject_PreviewMouseMove));
		}

		// Token: 0x060001F6 RID: 502 RVA: 0x0000A3DD File Offset: 0x000085DD
		private void Associated_PreviewKeyUp(object sender, KeyEventArgs e)
		{
			if (e.Key == Key.Escape)
			{
				this.ClearAdorner();
			}
		}

		// Token: 0x060001F7 RID: 503 RVA: 0x0000A3F0 File Offset: 0x000085F0
		private XmlElement GetTarget(XmlElement target, AdornedControl control)
		{
			if (target == null)
			{
				return target;
			}
			if (target.Parent != null)
			{
				switch (target.Parent.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					return target.Parent;
				}
			}
			return target;
		}

		// Token: 0x060001F8 RID: 504 RVA: 0x0000A43C File Offset: 0x0000863C
		private void AssociatedObject_PreviewMouseMove(object sender, MouseEventArgs e)
		{
			if (e.LeftButton != MouseButtonState.Pressed)
			{
				return;
			}
			Point position = e.GetPosition(this._container);
			if (Math.Abs(position.X - this._MouseDownStart.X) < (double)FormDesignSetting.UnitHeight && Math.Abs(position.Y - this._MouseDownStart.Y) < (double)FormDesignSetting.UnitHeight)
			{
				return;
			}
			this._MouseDownStart.X = (this._MouseDownStart.Y = 0.0);
			e.Handled = true;
			if (this._layer == null || this._parentDragFrom == null)
			{
				this.CreateAdorner();
			}
			if (this._moveOverModel != null)
			{
				this._moveOverModel.IsDragOver = false;
			}
			if (this._adorner != null && this._parentDragFrom != null)
			{
				if (this._adorner.Visibility == Visibility.Collapsed)
				{
					this._adorner.Visibility = Visibility.Visible;
				}
				this._adorner.UpdatePosition(position);
				AdornedControl parentByPoint = VisualTreeHelperEx.GetParentByPoint<AdornedControl>(this._container, position);
				if (parentByPoint == null)
				{
					return;
				}
				XmlElement xmlElement = parentByPoint.DataContext as XmlElement;
				if (xmlElement != null)
				{
					ComponentType type = xmlElement.Type;
					if (type == ComponentType.Folder)
					{
						TabItem parentByPoint2 = VisualTreeHelperEx.GetParentByPoint<TabItem>(this._container, position);
						if (parentByPoint2 != null && parentByPoint2.DataContext is XmlElement)
						{
							this._moveOverModel = parentByPoint2.DataContext as XmlElement;
							this._moveOverModel.IsFocused = true;
						}
						else
						{
							this._moveOverModel = xmlElement.Nodes.Where<XmlElement>((XmlElement t) => t.IsFocused).FirstOrDefault<XmlElement>();
						}
					}
					else
					{
						this._moveOverModel = this.GetTarget(parentByPoint.DataContext as XmlElement, parentByPoint);
					}
				}
				switch (this._parentDragFrom.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					if (this._parentDragFrom != this._moveOverModel)
					{
						this._moveOverModel = null;
					}
					break;
				}
				if (this._moveOverModel != null)
				{
					switch (this._moveOverModel.Type)
					{
					case ComponentType.Table:
					case ComponentType.Tree:
						if (this._parentDragFrom != this._moveOverModel)
						{
							this._moveOverModel = null;
						}
						break;
					}
				}
				if (this._moveOverModel != null)
				{
					foreach (XmlElement xmlElement2 in this._selectedComponents)
					{
						if (xmlElement2.IsCantMove || !ComponentFactory.AcceptMimes(this._moveOverModel, xmlElement2))
						{
							this._moveOverModel = null;
							break;
						}
					}
				}
				if (this._moveOverModel != null)
				{
					this._moveOverModel.IsDragOver = true;
				}
				this._moveOverObject = ((this._moveOverModel == null) ? null : parentByPoint);
				if (this._moveOverObject == null)
				{
					if (this._container != null)
					{
						this._container.Cursor = ((this._denyCursor == null) ? Cursors.No : this._denyCursor);
						return;
					}
				}
				else if (this._container != null)
				{
					this._container.Cursor = this._addCursor;
				}
			}
		}

		// Token: 0x060001F9 RID: 505 RVA: 0x0000A73C File Offset: 0x0000893C
		private void AssociatedObject_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			if (this._moveOverObject != null && this._dragDropCommand != null)
			{
				Point position = e.GetPosition(this._moveOverObject);
				int num = (int)position.X;
				int num2 = (int)position.Y;
				this._container.Cursor = null;
				XmlElement target = this.GetTarget(this._moveOverObject.DataContext as XmlElement, this._moveOverObject);
				if (target == null)
				{
					this.ClearAdorner();
					return;
				}
				switch (target.Type)
				{
				case ComponentType.Folder:
				{
					int num3 = -1;
					foreach (XmlElement xmlElement in target.Nodes)
					{
						if (xmlElement.IsFocused)
						{
							num3 = xmlElement.Index;
							break;
						}
					}
					if (this._dragDropCommand.IsMovePages)
					{
						this._dragDropCommand.SetDestination(target, num3);
						goto IL_023B;
					}
					this._dragDropCommand.SetDestination(target.Nodes[num3], num, num2);
					goto IL_023B;
				}
				case ComponentType.HBox:
					this._dragDropCommand.SetDestination(target, (position.X <= (double)(FormDesignSetting.UnitWidth * 2)) ? 0 : target.Nodes.Count);
					goto IL_023B;
				case ComponentType.Page:
					if (this._dragDropCommand.IsMovePages)
					{
						int index = target.Index;
						this._dragDropCommand.SetDestination(target.Parent, index);
						goto IL_023B;
					}
					this._dragDropCommand.SetDestination(target, 0);
					this._dragDropCommand.SetDestination(target, num, num2);
					goto IL_023B;
				case ComponentType.Table:
				case ComponentType.Tree:
				{
					XmlElement xmlElement2 = (this._moveOverObject.Content as FrameworkElement).DataContext as XmlElement;
					if (xmlElement2.Type == ComponentType.Table)
					{
						this._dragDropCommand.SetDestination(target, xmlElement2.Nodes.Count);
						goto IL_023B;
					}
					this._dragDropCommand.SetDestination(target, xmlElement2.Index);
					goto IL_023B;
				}
				case ComponentType.VBox:
					this._dragDropCommand.SetDestination(target, (position.Y <= (double)FormDesignSetting.UnitHeight) ? 0 : target.Nodes.Count);
					goto IL_023B;
				}
				this._dragDropCommand.SetDestination(target, num, num2);
				IL_023B:
				this._dragDropCommand.Execute();
				this._dragDropCommand = null;
			}
			this.ClearAdorner();
		}

		// Token: 0x060001FA RID: 506 RVA: 0x0000A9AC File Offset: 0x00008BAC
		private void CreateAdorner()
		{
			List<XmlElement> list = new List<XmlElement>();
			if (ComponentHelper.Get(this._key).SelectedObjects.Count<XmlElement>() == 0)
			{
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this._key).SelectedObjects)
			{
				if (xmlElement.IsCantMove)
				{
					DesignerMessageBox.Show("包含不可移動的物件", "Warning", MessageBoxButton.OK, MessageBoxImage.Exclamation);
					return;
				}
				if (!list.Contains(xmlElement))
				{
					list.Add(xmlElement);
					if (xmlElement.BindElement != null)
					{
						list.Add(xmlElement.BindElement);
					}
				}
			}
			ComponentHelper.Get(this._key).MultipleSelection(list);
			this._layer = AdornerLayer.GetAdornerLayer(this._container);
			if (this._componentsHost == null)
			{
				this._componentsHost = new WidgetAdornerView();
			}
			this._selectedComponents.Clear();
			this._parentDragFrom = ComponentHelper.Get(this._key).SelectedObjects.FirstOrDefault<XmlElement>().Parent;
			this.CreateVisualComponent();
			if (this._adorner == null)
			{
				this._adorner = new VisualComponentsAdorner(this._container, this._componentsHost);
			}
			this._layer.Visibility = Visibility.Visible;
			this._layer.Add(this._adorner);
			this._container.Cursor = this._addCursor;
		}

		// Token: 0x060001FB RID: 507 RVA: 0x0000AB24 File Offset: 0x00008D24
		private void CreateVisualComponent()
		{
			if (this._parentDragFrom == null)
			{
				return;
			}
			List<XmlElement> list = ComponentHelper.Get(this._key).SelectedObjects.ToList<XmlElement>();
			ComponentHelper.Get(this._key).ClearSelection();
			this._dragDropCommand = new DragComponentsUndoRedoCommand(this._parentDragFrom);
			int num = int.MaxValue;
			int num2 = int.MaxValue;
			ComponentType type = this._parentDragFrom.Type;
			if (type != ComponentType.Folder)
			{
				switch (type)
				{
				case ComponentType.HBox:
				case ComponentType.Table:
				case ComponentType.Tree:
				case ComponentType.VBox:
					break;
				case ComponentType.Page:
				case ComponentType.RadioGroup:
				case ComponentType.ScrollGrid:
					goto IL_00A8;
				default:
					goto IL_00A8;
				}
			}
			list.Sort((XmlElement a, XmlElement b) => a.Index - b.Index);
			IL_00A8:
			foreach (XmlElement xmlElement in list)
			{
				num = Math.Min(num, xmlElement.GridX);
				num2 = Math.Min(num2, xmlElement.GridY);
			}
			foreach (XmlElement xmlElement2 in list)
			{
				this._dragDropCommand.AppendSelection(xmlElement2, xmlElement2.Index, (int.MaxValue == num) ? xmlElement2.GridX : num, (int.MaxValue == num2) ? xmlElement2.GridY : num2);
			}
			foreach (XmlElement xmlElement3 in list)
			{
				xmlElement3.Parent.RemoveNode(xmlElement3, false);
				xmlElement3.GridX -= ((int.MaxValue == num) ? xmlElement3.GridX : num);
				xmlElement3.GridY -= ((int.MaxValue == num2) ? xmlElement3.GridY : num2);
				this._selectedComponents.Add(xmlElement3);
			}
			this._componentsHost.DataContext = this._selectedComponents;
		}

		// Token: 0x060001FC RID: 508 RVA: 0x0000AD44 File Offset: 0x00008F44
		private void ClearAdorner()
		{
			if (this._container != null)
			{
				this._container.Cursor = null;
			}
			this._moveOverObject = null;
			if (this._moveOverModel != null)
			{
				this._moveOverModel.IsDragOver = false;
			}
			if (this._layer != null && this._adorner != null)
			{
				this._layer.Remove(this._adorner);
			}
			this._layer = null;
			this._adorner = null;
			if (this._componentsHost != null)
			{
				this._componentsHost.DataContext = Binding.DoNothing;
			}
			this._componentsHost = null;
			this._parentDragFrom = null;
			this._selectedComponents.Clear();
			base.AssociatedObject.RemoveHandler(UIElement.MouseMoveEvent, new MouseEventHandler(this.AssociatedObject_PreviewMouseMove));
			if (this._dragDropCommand != null)
			{
				this._dragDropCommand.Undo();
				this._dragDropCommand.Clear();
			}
			this._dragDropCommand = null;
		}

		// Token: 0x04000115 RID: 277
		private PackageKey _key;

		// Token: 0x04000116 RID: 278
		private FrameworkElement _container;

		// Token: 0x04000117 RID: 279
		private WidgetAdornerView _componentsHost;

		// Token: 0x04000118 RID: 280
		private AdornerLayer _layer;

		// Token: 0x04000119 RID: 281
		private VisualComponentsAdorner _adorner;

		// Token: 0x0400011A RID: 282
		private Cursor _addCursor;

		// Token: 0x0400011B RID: 283
		private Cursor _denyCursor;

		// Token: 0x0400011C RID: 284
		private AdornedControl _moveOverObject;

		// Token: 0x0400011D RID: 285
		private XmlElement _moveOverModel;

		// Token: 0x0400011E RID: 286
		private List<XmlElement> _selectedComponents;

		// Token: 0x0400011F RID: 287
		private XmlElement _parentDragFrom;

		// Token: 0x04000120 RID: 288
		private DragComponentsUndoRedoCommand _dragDropCommand;

		// Token: 0x04000121 RID: 289
		private Point _MouseDownStart = new Point(0.0, 0.0);
	}
}
