using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using System.Windows.Shapes;
using SpecDesigner.FormEditor.Helpers;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000012 RID: 18
	public partial class FormStructureViewer : UserControl
	{
		// Token: 0x06000056 RID: 86 RVA: 0x00003B8C File Offset: 0x00001D8C
		public FormStructureViewer()
		{
			this.InitializeComponent();
			base.DataContextChanged += this.FormStructureViewer_DataContextChanged;
			base.AddHandler(UIElement.MouseLeftButtonUpEvent, new RoutedEventHandler(this.FormStructureViewer_MouseLeftButtonUp), true);
			base.AddHandler(UIElement.MouseLeftButtonDownEvent, new RoutedEventHandler(this.FormStructureViewer_MouseLeftButtonDown), true);
			base.AddHandler(UIElement.MouseMoveEvent, new RoutedEventHandler(this.FormStructureViewer_MouseMove), true);
			base.AddHandler(UIElement.KeyUpEvent, new KeyEventHandler(this.FormStructureViewer_KeyUp), true);
			this.BindCommands();
		}

		// Token: 0x06000057 RID: 87 RVA: 0x00003C20 File Offset: 0x00001E20
		private void BindCommands()
		{
			base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Undo, new ExecutedRoutedEventHandler(this.OnUndo), new CanExecuteRoutedEventHandler(this.CanUndo)));
			base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Redo, new ExecutedRoutedEventHandler(this.OnRedo), new CanExecuteRoutedEventHandler(this.CanRedo)));
			base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Delete, new ExecutedRoutedEventHandler(this.ExecuteDelete), new CanExecuteRoutedEventHandler(this.CanDelete)));
			base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Copy, new ExecutedRoutedEventHandler(this.ExecuteCopy), new CanExecuteRoutedEventHandler(this.CanCopy)));
			base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Paste, new ExecutedRoutedEventHandler(this.ExecutePaste), new CanExecuteRoutedEventHandler(this.CanPaste)));
		}

		// Token: 0x06000058 RID: 88 RVA: 0x00003D13 File Offset: 0x00001F13
		private void CanUndo(object target, CanExecuteRoutedEventArgs args)
		{
			if (this._programKey == null)
			{
				args.CanExecute = false;
				return;
			}
			if (SettingManager.Get().GetUndoRedoManager(this._programKey).UndoCount > 0)
			{
				args.CanExecute = true;
				return;
			}
			args.CanExecute = false;
		}

		// Token: 0x06000059 RID: 89 RVA: 0x00003D52 File Offset: 0x00001F52
		private void OnUndo(object target, ExecutedRoutedEventArgs args)
		{
			if (this._programKey == null)
			{
				return;
			}
			SettingManager.Get().GetUndoRedoManager(this._programKey).Undo();
		}

		// Token: 0x0600005A RID: 90 RVA: 0x00003D78 File Offset: 0x00001F78
		private void CanRedo(object target, CanExecuteRoutedEventArgs args)
		{
			if (this._programKey == null)
			{
				args.CanExecute = false;
				return;
			}
			if (SettingManager.Get().GetUndoRedoManager(this._programKey).RedoCount > 0)
			{
				args.CanExecute = true;
				return;
			}
			args.CanExecute = false;
		}

		// Token: 0x0600005B RID: 91 RVA: 0x00003DB7 File Offset: 0x00001FB7
		private void OnRedo(object target, ExecutedRoutedEventArgs args)
		{
			if (this._programKey == null)
			{
				return;
			}
			SettingManager.Get().GetUndoRedoManager(this._programKey).Redo();
		}

		// Token: 0x0600005C RID: 92 RVA: 0x00003DDD File Offset: 0x00001FDD
		private void CanDelete(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.CanDelete(ManagedForm.Current, e);
			}
		}

		// Token: 0x0600005D RID: 93 RVA: 0x00003DF6 File Offset: 0x00001FF6
		private void ExecuteDelete(object sender, ExecutedRoutedEventArgs e)
		{
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.ExecuteDelete(ManagedForm.Current, e);
			}
		}

		// Token: 0x0600005E RID: 94 RVA: 0x00003E0F File Offset: 0x0000200F
		private void CanCopy(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.CanCopy(ManagedForm.Current, e);
			}
		}

		// Token: 0x0600005F RID: 95 RVA: 0x00003E28 File Offset: 0x00002028
		private void ExecuteCopy(object sender, ExecutedRoutedEventArgs e)
		{
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.ExecuteCopy(ManagedForm.Current, e);
			}
		}

		// Token: 0x06000060 RID: 96 RVA: 0x00003E41 File Offset: 0x00002041
		private void CanPaste(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.CanPaste(ManagedForm.Current, e);
			}
		}

		// Token: 0x06000061 RID: 97 RVA: 0x00003E5A File Offset: 0x0000205A
		private void ExecutePaste(object sender, ExecutedRoutedEventArgs e)
		{
			if (ManagedForm.Current != null)
			{
				ManagedForm.Current.ExecutePaste(ManagedForm.Current, e);
			}
		}

		// Token: 0x06000062 RID: 98 RVA: 0x00003E73 File Offset: 0x00002073
		private void BtnExpandAll_Click(object sender, RoutedEventArgs e)
		{
			XmlElement.ExpandAll(this.UITree.DataContext as XmlElement, true);
		}

		// Token: 0x06000063 RID: 99 RVA: 0x00003E8B File Offset: 0x0000208B
		private void BtnCollapseAll_Click(object sender, RoutedEventArgs e)
		{
			XmlElement.ExpandAll(this.UITree.DataContext as XmlElement, false);
		}

		// Token: 0x06000064 RID: 100 RVA: 0x00003EA4 File Offset: 0x000020A4
		private void CancelDnD()
		{
			this._isMouseDown = false;
			if (this.draggedItem != null)
			{
				this.draggedItem.Visibility = Visibility.Visible;
			}
			this.draggedItem = null;
			this.targetItem = null;
			base.ReleaseMouseCapture();
			this.hideCursor();
			base.Cursor = null;
			this.shadowCursor.Source = null;
			this._dropDirection = DropDirection.None;
		}

		// Token: 0x06000065 RID: 101 RVA: 0x00003F00 File Offset: 0x00002100
		private bool CheckDropTarget(XmlElement targetComponent, XmlElement draggedComponent)
		{
			if (targetComponent == null || draggedComponent == null)
			{
				return false;
			}
			switch (this._dropDirection)
			{
			case DropDirection.Top:
			case DropDirection.Bottom:
				targetComponent = targetComponent.Parent;
				break;
			}
			if (targetComponent == null || !FormDesignSetting.IsContainer(targetComponent.NodeName))
			{
				return false;
			}
			if (targetComponent != null)
			{
				switch (targetComponent.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					if (targetComponent != draggedComponent.Parent)
					{
						targetComponent = null;
					}
					break;
				}
			}
			if (draggedComponent.Parent != null)
			{
				switch (draggedComponent.Parent.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					if (targetComponent != draggedComponent.Parent)
					{
						targetComponent = null;
					}
					break;
				}
			}
			return targetComponent != null && ComponentFactory.AcceptMimes(targetComponent, draggedComponent.Type);
		}

		// Token: 0x06000066 RID: 102 RVA: 0x00003FB8 File Offset: 0x000021B8
		private BitmapSource createShadowImageSource()
		{
			TreeViewItem treeViewItem = this.draggedItem;
			if (treeViewItem == null)
			{
				return null;
			}
			Border border = treeViewItem.Template.FindName("Bd", treeViewItem) as Border;
			Rect descendantBounds = VisualTreeHelper.GetDescendantBounds(border);
			RenderTargetBitmap renderTargetBitmap = new RenderTargetBitmap((int)descendantBounds.Width, (int)descendantBounds.Height, 96.0, 96.0, PixelFormats.Pbgra32);
			renderTargetBitmap.Render(border);
			renderTargetBitmap.Freeze();
			return renderTargetBitmap;
		}

		// Token: 0x06000067 RID: 103 RVA: 0x0000402A File Offset: 0x0000222A
		private void FormStructureViewer_DataContextChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			if (e.NewValue != null && e.NewValue is XmlElement)
			{
				this.CancelDnD();
				this._programKey = (e.NewValue as XmlElement).Key;
				return;
			}
			this._programKey = null;
		}

		// Token: 0x06000068 RID: 104 RVA: 0x00004068 File Offset: 0x00002268
		private void FormStructureViewer_KeyUp(object sender, KeyEventArgs e)
		{
			if (e.Key == Key.Escape)
			{
				this.CancelDnD();
			}
		}

		// Token: 0x06000069 RID: 105 RVA: 0x0000407C File Offset: 0x0000227C
		private void GetAllNodes(XmlElement xe, ICollection<XmlElement> nodes)
		{
			if (xe != null)
			{
				if (nodes != null)
				{
					nodes.Add(xe);
				}
				foreach (XmlElement xmlElement in xe.Nodes)
				{
					this.GetAllNodes(xmlElement, nodes);
				}
			}
		}

		// Token: 0x0600006A RID: 106 RVA: 0x000040D8 File Offset: 0x000022D8
		private void FormStructureViewer_MouseLeftButtonDown(object sender, RoutedEventArgs e)
		{
			FrameworkElement frameworkElement = e.OriginalSource as FrameworkElement;
			if (frameworkElement != null && (frameworkElement.Name.Equals(this.ImgExpandAll.Name) || frameworkElement.Name.Equals(this.ImgCollapseAll.Name)))
			{
				return;
			}
			this._lastMouseDown = Mouse.GetPosition(this);
			IInputElement inputElement = this.UITree.InputHitTest(Mouse.GetPosition(this));
			this.draggedItem = VisualTreeHelperEx.FindVisualParent<TreeViewItem>(inputElement as DependencyObject);
			if (this.draggedItem != null)
			{
				this._lastItem = this._currentItem;
				this._currentItem = this.draggedItem.DataContext as XmlElement;
				if ((Keyboard.Modifiers & ModifierKeys.Shift) == ModifierKeys.Shift)
				{
					if (this._shiftSelectNodes == null)
					{
						this._shiftSelectNodes = new List<XmlElement>();
					}
					if (this._shiftSelectNodes.Count == 0)
					{
						this._shiftSelectNodes.Add(this._lastItem);
					}
					this._shiftSelectNodes.Add(this._currentItem);
				}
				else if (this._shiftSelectNodes != null)
				{
					this._shiftSelectNodes.Clear();
				}
				if (this._shiftSelectNodes != null && this._shiftSelectNodes.Count >= 1)
				{
					bool flag = false;
					if (this._shiftSelectNodes[0] == this._shiftSelectNodes[this._shiftSelectNodes.Count - 1])
					{
						ComponentHelper.Get(this._shiftSelectNodes[0].Key).AddSelection(this._shiftSelectNodes[0], true);
						return;
					}
					ICollection<XmlElement> collection = new List<XmlElement>();
					this.GetAllNodes(this.UITree.DataContext as XmlElement, collection);
					IList<XmlElement> list = new List<XmlElement>();
					foreach (XmlElement xmlElement in collection)
					{
						if (xmlElement == this._shiftSelectNodes[0] || xmlElement == this._shiftSelectNodes[this._shiftSelectNodes.Count - 1])
						{
							flag = !flag;
							if (xmlElement.Parent == this._shiftSelectNodes[0].Parent)
							{
								list.Add(xmlElement);
							}
						}
						else if (flag && xmlElement.Parent == this._shiftSelectNodes[0].Parent)
						{
							list.Add(xmlElement);
						}
					}
					ComponentHelper.Get(this._programKey).MultipleSelection(list);
					return;
				}
				else
				{
					if (!this.CanDrag())
					{
						return;
					}
					this._isMouseDown = true;
					base.CaptureMouse();
				}
			}
		}

		// Token: 0x0600006B RID: 107 RVA: 0x0000435C File Offset: 0x0000255C
		private bool CanDrag()
		{
			XmlElement xmlElement = this.draggedItem.DataContext as XmlElement;
			return xmlElement != null && xmlElement.Parent != null && !xmlElement.IsCantMove;
		}

		// Token: 0x0600006C RID: 108 RVA: 0x00004390 File Offset: 0x00002590
		private void DoMultiDrag(DragComponentsUndoRedoCommand command, XmlElement target, XmlElement source)
		{
			if (target == null || source == null)
			{
				return;
			}
			int num = 0;
			XmlElement xmlElement = null;
			switch (this._dropDirection)
			{
			case DropDirection.Top:
				xmlElement = target.Parent;
				num = target.Index;
				break;
			case DropDirection.Bottom:
				xmlElement = target.Parent;
				num = target.Index + 1;
				break;
			case DropDirection.Center:
				xmlElement = target;
				num = 0;
				break;
			}
			if (xmlElement == null)
			{
				return;
			}
			command.AppendSelection(source, source.Index);
			switch (xmlElement.Type)
			{
			case ComponentType.Table:
			case ComponentType.Tree:
				command.SetDestination(xmlElement, num);
				return;
			default:
				command.SetDestination(xmlElement, 0, 0);
				return;
			}
		}

		// Token: 0x0600006D RID: 109 RVA: 0x00004428 File Offset: 0x00002628
		private void FormStructureViewer_MouseLeftButtonUp(object sender, RoutedEventArgs e)
		{
			List<XmlElement> list = new List<XmlElement>(ComponentHelper.Get(this._programKey).SelectedObjects);
			if (list.Count > 1)
			{
				if (this.targetItem != null && this._isMouseDown)
				{
					XmlElement xmlElement = this.targetItem.DataContext as XmlElement;
					if (xmlElement == null)
					{
						return;
					}
					DragComponentsUndoRedoCommand dragComponentsUndoRedoCommand = new DragComponentsUndoRedoCommand(list[0].Parent);
					bool flag = false;
					foreach (XmlElement xmlElement2 in list)
					{
						if (this.CheckDropTarget(xmlElement, xmlElement2))
						{
							this.DoMultiDrag(dragComponentsUndoRedoCommand, xmlElement, xmlElement2);
							flag = true;
						}
					}
					if (flag)
					{
						dragComponentsUndoRedoCommand.Execute();
					}
					if (this._shiftSelectNodes != null)
					{
						this._shiftSelectNodes.Clear();
					}
					list.Clear();
					list = null;
				}
				else if (this.targetItem == null && this._isMouseDown && (Keyboard.Modifiers & ModifierKeys.Control) != ModifierKeys.Control)
				{
					ComponentHelper.Get(this._programKey).ClearSelection();
					ComponentHelper.Get(this._programKey).AddSelection(this.draggedItem.DataContext as XmlElement, true);
				}
			}
			else if (this.targetItem != null && this._isMouseDown && this.CheckDropTarget(this.targetItem.DataContext as XmlElement, this.draggedItem.DataContext as XmlElement))
			{
				XmlElement xmlElement3 = this.draggedItem.DataContext as XmlElement;
				XmlElement xmlElement4 = this.targetItem.DataContext as XmlElement;
				if (xmlElement4 == null)
				{
					return;
				}
				int num = 0;
				XmlElement xmlElement5 = null;
				switch (this._dropDirection)
				{
				case DropDirection.Top:
					xmlElement5 = xmlElement4.Parent;
					num = xmlElement4.Index;
					break;
				case DropDirection.Bottom:
					xmlElement5 = xmlElement4.Parent;
					num = xmlElement4.Index + 1;
					break;
				case DropDirection.Center:
					xmlElement5 = xmlElement4;
					num = 0;
					break;
				}
				if (xmlElement5 == null)
				{
					return;
				}
				DragComponentsUndoRedoCommand dragComponentsUndoRedoCommand2 = new DragComponentsUndoRedoCommand(xmlElement3.Parent);
				dragComponentsUndoRedoCommand2.AppendSelection(xmlElement3, xmlElement3.Index);
				switch (xmlElement5.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					dragComponentsUndoRedoCommand2.SetDestination(xmlElement5, num);
					break;
				default:
					dragComponentsUndoRedoCommand2.SetDestination(xmlElement5, 0, 0);
					break;
				}
				dragComponentsUndoRedoCommand2.Execute();
			}
			this.CancelDnD();
		}

		// Token: 0x0600006E RID: 110 RVA: 0x00004690 File Offset: 0x00002890
		private void FormStructureViewer_MouseMove(object sender, RoutedEventArgs e)
		{
			Point position = Mouse.GetPosition(this);
			if (this._isMouseDown && (Math.Abs(position.X - this._lastMouseDown.X) > 10.0 || Math.Abs(position.Y - this._lastMouseDown.Y) > 10.0))
			{
				this.UpdateShadowCursor(position);
				IInputElement inputElement = this.UITree.InputHitTest(position);
				if (inputElement == null)
				{
					return;
				}
				this.targetItem = VisualTreeHelperEx.FindVisualParent<TreeViewItem>(inputElement as DependencyObject);
				if (this.draggedItem != null && this.draggedItem != this.targetItem && this.targetItem != null)
				{
					this.UpdateDropDirection(this.targetItem);
					bool flag = this.CheckDropTarget(this.targetItem.DataContext as XmlElement, this.draggedItem.DataContext as XmlElement);
					this.UpdateCursorPosition(this.targetItem, flag);
					List<XmlElement> list = new List<XmlElement>(ComponentHelper.Get(this._programKey).SelectedObjects);
					if (list.Count <= 1 && flag)
					{
						this.draggedItem.Visibility = Visibility.Collapsed;
					}
					list.Clear();
				}
			}
		}

		// Token: 0x0600006F RID: 111 RVA: 0x000047B8 File Offset: 0x000029B8
		private void UpdateDropDirection(TreeViewItem treeViewItem)
		{
			double num = 5.0;
			Border border = treeViewItem.Template.FindName("Bd", treeViewItem) as Border;
			if (border != null)
			{
				Point position = Mouse.GetPosition(border);
				double height = border.DesiredSize.Height;
				if (position.Y < num)
				{
					this._dropDirection = DropDirection.Top;
				}
				else if (position.Y > height - num)
				{
					this._dropDirection = DropDirection.Bottom;
				}
				else
				{
					this._dropDirection = DropDirection.Center;
				}
			}
			List<XmlElement> list = new List<XmlElement>(ComponentHelper.Get(this._programKey).SelectedObjects);
			if (list.Count > 1 && this._dropDirection != DropDirection.Center)
			{
				this._dropDirection = DropDirection.Center;
			}
		}

		// Token: 0x06000070 RID: 112 RVA: 0x00004860 File Offset: 0x00002A60
		private void hideCursor()
		{
			this.adornerElement.Visibility = Visibility.Collapsed;
			this.adornerElement.Width = 0.0;
			this.adornerElement.Height = 0.0;
		}

		// Token: 0x06000071 RID: 113 RVA: 0x00004898 File Offset: 0x00002A98
		private void UpdateCursorPosition(TreeViewItem treeViewItem, bool canDrop)
		{
			if (treeViewItem == null)
			{
				this.adornerElement.Visibility = Visibility.Collapsed;
				return;
			}
			Border border = treeViewItem.Template.FindName("Bd", treeViewItem) as Border;
			if (border != null)
			{
				Point point = border.TransformToVisual(this.UITree).Transform(new Point(0.0, 0.0));
				double height = border.DesiredSize.Height;
				switch (this._dropDirection)
				{
				case DropDirection.Top:
					Canvas.SetTop(this.adornerElement, point.Y);
					Canvas.SetLeft(this.adornerElement, point.X);
					this.adornerElement.Width = border.DesiredSize.Width + 4.0;
					this.adornerElement.Height = 2.0;
					break;
				case DropDirection.Bottom:
					Canvas.SetTop(this.adornerElement, point.Y + height - 2.0);
					Canvas.SetLeft(this.adornerElement, point.X);
					this.adornerElement.Width = border.DesiredSize.Width + 4.0;
					this.adornerElement.Height = 2.0;
					break;
				case DropDirection.Center:
					Canvas.SetTop(this.adornerElement, point.Y);
					Canvas.SetLeft(this.adornerElement, point.X);
					this.adornerElement.Width = border.DesiredSize.Width + 4.0;
					this.adornerElement.Height = border.DesiredSize.Height;
					break;
				}
			}
			this.adornerElement.Visibility = (canDrop ? Visibility.Visible : Visibility.Collapsed);
			base.Cursor = (canDrop ? null : Cursors.No);
		}

		// Token: 0x06000072 RID: 114 RVA: 0x00004A7C File Offset: 0x00002C7C
		private void UpdateShadowCursor(Point currentPosition)
		{
			double num = 60.0;
			double y = currentPosition.Y;
			double x = currentPosition.X;
			double num2 = 5.0;
			ScrollViewer scrollViewer = this.UITree.Template.FindName("_tv_scrollviewer_", this.UITree) as ScrollViewer;
			if (y < num)
			{
				scrollViewer.ScrollToVerticalOffset(scrollViewer.VerticalOffset - num2);
			}
			else if (y > this.UITree.ActualHeight - num)
			{
				scrollViewer.ScrollToVerticalOffset(scrollViewer.VerticalOffset + num2);
			}
			if (x < num)
			{
				scrollViewer.ScrollToHorizontalOffset(scrollViewer.HorizontalOffset - num2);
			}
			else if (x > this.UITree.ActualWidth - num)
			{
				scrollViewer.ScrollToHorizontalOffset(scrollViewer.HorizontalOffset + num2);
			}
			if (this.shadowCursor.Source == null)
			{
				this.shadowCursor.Source = this.createShadowImageSource();
			}
			Canvas.SetTop(this.shadowCursor, currentPosition.Y + 12.0);
			Canvas.SetLeft(this.shadowCursor, currentPosition.X - 5.0);
		}

		// Token: 0x14000001 RID: 1
		// (add) Token: 0x06000073 RID: 115 RVA: 0x00004B94 File Offset: 0x00002D94
		// (remove) Token: 0x06000074 RID: 116 RVA: 0x00004BCC File Offset: 0x00002DCC
		public event RoutedEventHandler FormItemSelected;

		// Token: 0x06000075 RID: 117 RVA: 0x00004C04 File Offset: 0x00002E04
		private void UITreeItem_Selected(object sender, RoutedEventArgs e)
		{
			TreeViewItem treeViewItem = (TreeViewItem)e.OriginalSource;
			int childrenCount = VisualTreeHelper.GetChildrenCount(treeViewItem);
			for (int i = childrenCount - 1; i >= 0; i--)
			{
				DependencyObject child = VisualTreeHelper.GetChild(treeViewItem, i);
				((FrameworkElement)child).BringIntoView();
			}
			if (this.FormItemSelected != null)
			{
				object header = treeViewItem.Header;
				if (header != null)
				{
					this.FormItemSelected(header, e);
				}
			}
		}

		// Token: 0x0400003F RID: 63
		private bool _isMouseDown;

		// Token: 0x04000040 RID: 64
		private Point _lastMouseDown;

		// Token: 0x04000041 RID: 65
		private PackageKey _programKey;

		// Token: 0x04000042 RID: 66
		private TreeViewItem draggedItem;

		// Token: 0x04000043 RID: 67
		private TreeViewItem targetItem;

		// Token: 0x04000044 RID: 68
		private DropDirection _dropDirection;

		// Token: 0x04000045 RID: 69
		private IList<XmlElement> _shiftSelectNodes;

		// Token: 0x04000046 RID: 70
		private XmlElement _currentItem;

		// Token: 0x04000047 RID: 71
		private XmlElement _lastItem;
	}
}
