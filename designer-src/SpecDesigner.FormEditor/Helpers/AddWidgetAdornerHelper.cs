using System;
using System.Collections.Generic;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Documents;
using System.Windows.Input;
using SpecDesigner.Controls.AdornedControl;
using SpecDesigner.FormEditor.Views;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Helpers
{
	// Token: 0x0200000C RID: 12
	public static class AddWidgetAdornerHelper
	{
		// Token: 0x0600002F RID: 47 RVA: 0x00002CB4 File Offset: 0x00000EB4
		public static void Show(UIElement container, ComponentType widgetType, PackageKey programKey)
		{
			List<XmlElement> list = ComponentFactory.CreateEmptyComponentWithLabel(programKey, widgetType);
			if (list == null)
			{
				return;
			}
			AddWidgetAdornerHelper.Show(container, list, true, SpecNodeType.NONE);
		}

		// Token: 0x06000030 RID: 48 RVA: 0x00002CD8 File Offset: 0x00000ED8
		public static void Show(UIElement container, SpecNodeType specType, PackageKey programKey)
		{
			XmlElement xmlElement = null;
			switch (specType)
			{
			case SpecNodeType.PROGREL:
			case SpecNodeType.REFERENCE:
			case SpecNodeType.MULTILANG:
				xmlElement = ComponentFactory.CreateEmptyComponent(programKey, specType);
				break;
			}
			if (xmlElement != null)
			{
				AddWidgetAdornerHelper.Show(container, new List<XmlElement> { xmlElement }, false, specType);
			}
		}

		// Token: 0x06000031 RID: 49 RVA: 0x00002D24 File Offset: 0x00000F24
		public static void Show(UIElement container, List<XmlElement> components, bool autoLayout, SpecNodeType specType = SpecNodeType.NONE)
		{
			AddWidgetAdornerHelper.clearAdorner();
			AddWidgetAdornerHelper._autoLayout = autoLayout;
			AddWidgetAdornerHelper._selectedComponents = components;
			if (AddWidgetAdornerHelper._autoLayout)
			{
				int num = 0;
				foreach (XmlElement xmlElement in AddWidgetAdornerHelper._selectedComponents)
				{
					if (xmlElement == null)
					{
						return;
					}
					if (num >= 0)
					{
						xmlElement.GridX = num;
					}
					num += xmlElement.GridWidth + 1;
				}
			}
			AddWidgetAdornerHelper._container = container;
			AddWidgetAdornerHelper._container.AddHandler(UIElement.PreviewMouseMoveEvent, new RoutedEventHandler(AddWidgetAdornerHelper.containerMouseMove), true);
			AddWidgetAdornerHelper._container.AddHandler(UIElement.PreviewMouseLeftButtonDownEvent, new RoutedEventHandler(AddWidgetAdornerHelper.containerMouseLeftButtonDown), true);
			AddWidgetAdornerHelper._layer = AdornerLayer.GetAdornerLayer(container);
			if (AddWidgetAdornerHelper._componentsHost == null)
			{
				AddWidgetAdornerHelper._componentsHost = new WidgetAdornerView();
			}
			AddWidgetAdornerHelper._componentsHost.DataContext = AddWidgetAdornerHelper._selectedComponents;
			Window.GetWindow(container).AddHandler(UIElement.KeyUpEvent, new KeyEventHandler(AddWidgetAdornerHelper.Window_KeyUp), true);
			AddWidgetAdornerHelper._adorner = new VisualComponentsAdorner(AddWidgetAdornerHelper._container, AddWidgetAdornerHelper._componentsHost);
			AddWidgetAdornerHelper._adorner.Visibility = Visibility.Collapsed;
			AddWidgetAdornerHelper._layer.Add(AddWidgetAdornerHelper._adorner);
			if (AddWidgetAdornerHelper._addCursor == null || AddWidgetAdornerHelper._denyCursor == null)
			{
				Uri uri = new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Add.cur", UriKind.Absolute);
				Uri uri2 = new Uri("pack://application:,,,/SpecDesigner.FormEditor;component/Stop.cur", UriKind.Absolute);
				AddWidgetAdornerHelper._addCursor = new Cursor(Application.GetResourceStream(uri).Stream);
				AddWidgetAdornerHelper._denyCursor = new Cursor(Application.GetResourceStream(uri2).Stream);
			}
			(container as FrameworkElement).Cursor = AddWidgetAdornerHelper._addCursor;
			AddWidgetAdornerHelper._specNodeType = specType;
		}

		// Token: 0x06000032 RID: 50 RVA: 0x00002EC4 File Offset: 0x000010C4
		private static void Window_KeyUp(object sender, KeyEventArgs e)
		{
			if (e.Key == Key.Escape)
			{
				AddWidgetAdornerHelper.clearAdorner();
			}
		}

		// Token: 0x06000033 RID: 51 RVA: 0x00002ED8 File Offset: 0x000010D8
		private static void containerMouseMove(object sender, RoutedEventArgs e)
		{
			if (AddWidgetAdornerHelper._adorner == null)
			{
				AddWidgetAdornerHelper.clearAdorner();
				return;
			}
			if (AddWidgetAdornerHelper._moveOverModel != null)
			{
				AddWidgetAdornerHelper._moveOverModel.IsDragOver = false;
			}
			Point position = (e as MouseEventArgs).GetPosition(AddWidgetAdornerHelper._container);
			if (AddWidgetAdornerHelper._adorner.Visibility == Visibility.Collapsed)
			{
				AddWidgetAdornerHelper._adorner.Visibility = Visibility.Visible;
			}
			AddWidgetAdornerHelper._adorner.UpdatePosition(position);
			AdornedControl parentByPoint = VisualTreeHelperEx.GetParentByPoint<AdornedControl>(AddWidgetAdornerHelper._container, position);
			if (parentByPoint == null)
			{
				AddWidgetAdornerHelper._moveOverObject = null;
				return;
			}
			AddWidgetAdornerHelper._moveOverModel = AddWidgetAdornerHelper.GetTarget(parentByPoint.DataContext as XmlElement, parentByPoint);
			if (AddWidgetAdornerHelper._moveOverModel != null)
			{
				foreach (XmlElement xmlElement in AddWidgetAdornerHelper._selectedComponents)
				{
					if (AddWidgetAdornerHelper._moveOverModel == null)
					{
						break;
					}
					switch (AddWidgetAdornerHelper._moveOverModel.Type)
					{
					case ComponentType.Table:
					case ComponentType.Tree:
						if (xmlElement.Type == ComponentType.Label)
						{
							continue;
						}
						break;
					}
					if (!ComponentFactory.AcceptMimes(AddWidgetAdornerHelper._moveOverModel, xmlElement.Type))
					{
						AddWidgetAdornerHelper._moveOverModel = null;
						break;
					}
				}
			}
			AddWidgetAdornerHelper._moveOverObject = ((AddWidgetAdornerHelper._moveOverModel == null) ? null : parentByPoint);
			if (AddWidgetAdornerHelper._moveOverObject == null)
			{
				if (AddWidgetAdornerHelper._container != null)
				{
					(AddWidgetAdornerHelper._container as FrameworkElement).Cursor = ((AddWidgetAdornerHelper._denyCursor == null) ? Cursors.No : AddWidgetAdornerHelper._denyCursor);
					return;
				}
			}
			else
			{
				if (AddWidgetAdornerHelper._container != null)
				{
					(AddWidgetAdornerHelper._container as FrameworkElement).Cursor = AddWidgetAdornerHelper._addCursor;
				}
				AddWidgetAdornerHelper._moveOverModel.IsDragOver = true;
			}
		}

		// Token: 0x06000034 RID: 52 RVA: 0x0000305C File Offset: 0x0000125C
		private static XmlElement GetTarget(XmlElement target, AdornedControl control)
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
			if (AddWidgetAdornerHelper._selectedComponents.Count > 0 && AddWidgetAdornerHelper._selectedComponents[0].Type != ComponentType.Page && control.Content is TabControl)
			{
				TabControl tabControl = control.Content as TabControl;
				target = tabControl.SelectedContent as XmlElement;
				return target;
			}
			return target;
		}

		// Token: 0x06000035 RID: 53 RVA: 0x000030E8 File Offset: 0x000012E8
		private static void containerMouseLeftButtonDown(object sender, RoutedEventArgs e)
		{
			if (AddWidgetAdornerHelper._moveOverObject == null)
			{
				AddWidgetAdornerHelper.clearAdorner();
				return;
			}
			if (AddWidgetAdornerHelper._layer != null && AddWidgetAdornerHelper._layer.Visibility == Visibility.Visible && AddWidgetAdornerHelper._container != null)
			{
				Point position = (e as MouseEventArgs).GetPosition(AddWidgetAdornerHelper._moveOverObject);
				int num = FormDesignSetting.TransformToGridWidth(FormDesignSetting.GetNearWidth(position.X));
				int num2 = FormDesignSetting.TransformToGridHeight(FormDesignSetting.GetNearHeight(position.Y));
				(AddWidgetAdornerHelper._container as FrameworkElement).Cursor = null;
				XmlElement target = AddWidgetAdornerHelper.GetTarget(AddWidgetAdornerHelper._moveOverObject.DataContext as XmlElement, AddWidgetAdornerHelper._moveOverObject);
				if (target == null)
				{
					AddWidgetAdornerHelper.clearAdorner();
					return;
				}
				switch (target.Type)
				{
				case ComponentType.Group:
				case ComponentType.Page:
					num2--;
					break;
				}
				AddComponetsUndoRedoCommand addComponetsUndoRedoCommand;
				switch (target.Type)
				{
				case ComponentType.HBox:
					addComponetsUndoRedoCommand = ((Mouse.GetPosition(AddWidgetAdornerHelper._moveOverObject).X <= (double)(FormDesignSetting.UnitWidth * 2)) ? new AddComponetsUndoRedoCommand(AddWidgetAdornerHelper._selectedComponents, target) : new AddComponetsUndoRedoCommand(AddWidgetAdornerHelper._selectedComponents, target, target.Nodes.Count));
					goto IL_023D;
				case ComponentType.Table:
				case ComponentType.Tree:
				{
					List<XmlElement> list = new List<XmlElement>();
					foreach (XmlElement xmlElement in AddWidgetAdornerHelper._selectedComponents)
					{
						ComponentType type = xmlElement.Type;
						if (type != ComponentType.Label)
						{
							if (AddWidgetAdornerHelper._specNodeType == SpecNodeType.PROGREL || AddWidgetAdornerHelper._specNodeType == SpecNodeType.REFERENCE || AddWidgetAdornerHelper._specNodeType == SpecNodeType.MULTILANG)
							{
								list.Add(ComponentFactory.CreateEmptyComponentForBody(xmlElement.Key, AddWidgetAdornerHelper._specNodeType));
							}
							else
							{
								list.Add(xmlElement);
							}
						}
					}
					XmlElement xmlElement2 = AddWidgetAdornerHelper._moveOverObject.DataContext as XmlElement;
					addComponetsUndoRedoCommand = ((xmlElement2 == null) ? new AddComponetsUndoRedoCommand(list, target, num, num2) : new AddComponetsUndoRedoCommand(list, target, xmlElement2.Index));
					goto IL_023D;
				}
				case ComponentType.VBox:
					addComponetsUndoRedoCommand = ((Mouse.GetPosition(AddWidgetAdornerHelper._moveOverObject).Y <= (double)FormDesignSetting.UnitHeight) ? new AddComponetsUndoRedoCommand(AddWidgetAdornerHelper._selectedComponents, target) : new AddComponetsUndoRedoCommand(AddWidgetAdornerHelper._selectedComponents, target, target.Nodes.Count));
					goto IL_023D;
				}
				addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(AddWidgetAdornerHelper._selectedComponents, target, num, num2);
				IL_023D:
				addComponetsUndoRedoCommand.Execute();
			}
			AddWidgetAdornerHelper.clearAdorner();
		}

		// Token: 0x06000036 RID: 54 RVA: 0x00003350 File Offset: 0x00001550
		private static void AddNodes(PackageKey key, XmlElement element)
		{
			foreach (XmlElement xmlElement in element.Nodes)
			{
				AddWidgetAdornerHelper.AddNodes(key, xmlElement);
				SettingManager.Get().GetTzpManger(key).SpecificationInfo.Add(xmlElement);
			}
		}

		// Token: 0x06000037 RID: 55 RVA: 0x000033B4 File Offset: 0x000015B4
		private static void clearAdorner()
		{
			if (AddWidgetAdornerHelper._container != null)
			{
				AddWidgetAdornerHelper._container.RemoveHandler(UIElement.PreviewMouseMoveEvent, new RoutedEventHandler(AddWidgetAdornerHelper.containerMouseMove));
				AddWidgetAdornerHelper._container.RemoveHandler(UIElement.PreviewMouseLeftButtonDownEvent, new RoutedEventHandler(AddWidgetAdornerHelper.containerMouseLeftButtonDown));
				(AddWidgetAdornerHelper._container as FrameworkElement).Cursor = null;
				Window.GetWindow(AddWidgetAdornerHelper._container).RemoveHandler(UIElement.KeyUpEvent, new KeyEventHandler(AddWidgetAdornerHelper.Window_KeyUp));
			}
			if (AddWidgetAdornerHelper._moveOverModel != null)
			{
				AddWidgetAdornerHelper._moveOverModel.IsDragOver = false;
			}
			AddWidgetAdornerHelper._moveOverObject = null;
			if (AddWidgetAdornerHelper._layer != null && AddWidgetAdornerHelper._adorner != null)
			{
				AddWidgetAdornerHelper._layer.Remove(AddWidgetAdornerHelper._adorner);
			}
			AddWidgetAdornerHelper._layer = null;
			if (AddWidgetAdornerHelper._selectedComponents != null)
			{
				AddWidgetAdornerHelper._selectedComponents.Clear();
			}
			AddWidgetAdornerHelper._selectedComponents = null;
			AddWidgetAdornerHelper._container = null;
			AddWidgetAdornerHelper._adorner = null;
			if (AddWidgetAdornerHelper._componentsHost != null)
			{
				AddWidgetAdornerHelper._componentsHost.DataContext = Binding.DoNothing;
			}
			AddWidgetAdornerHelper._componentsHost = null;
		}

		// Token: 0x04000022 RID: 34
		private static UIElement _container;

		// Token: 0x04000023 RID: 35
		private static AdornerLayer _layer = null;

		// Token: 0x04000024 RID: 36
		private static VisualComponentsAdorner _adorner = null;

		// Token: 0x04000025 RID: 37
		private static WidgetAdornerView _componentsHost = null;

		// Token: 0x04000026 RID: 38
		private static List<XmlElement> _selectedComponents = null;

		// Token: 0x04000027 RID: 39
		private static Cursor _addCursor;

		// Token: 0x04000028 RID: 40
		private static Cursor _denyCursor;

		// Token: 0x04000029 RID: 41
		private static AdornedControl _moveOverObject = null;

		// Token: 0x0400002A RID: 42
		private static XmlElement _moveOverModel = null;

		// Token: 0x0400002B RID: 43
		private static bool _autoLayout = false;

		// Token: 0x0400002C RID: 44
		private static SpecNodeType _specNodeType = SpecNodeType.NONE;
	}
}
