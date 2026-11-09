using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interactivity;
using System.Windows.Markup;
using System.Windows.Media;
using System.Windows.Media.Animation;
using System.Windows.Threading;
using SpecDesigner.Controls.Controls;
using SpecDesigner.FormEditor.Behaviors;
using SpecDesigner.FormEditor.Helpers;
using SpecDesigner.FormEditor.Test;
using SpecDesignerCommon;
using SpecDesignerCommon.Connection;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.Logger;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x0200004F RID: 79
	public partial class ManagedForm : UserControl, IDisposable
	{
		// Token: 0x17000092 RID: 146
		// (get) Token: 0x060002E0 RID: 736 RVA: 0x0000EAD4 File Offset: 0x0000CCD4
		// (set) Token: 0x060002E1 RID: 737 RVA: 0x0000EADC File Offset: 0x0000CCDC
		[DefaultValue(false)]
		public bool IsSimpleForm { get; private set; }

		// Token: 0x17000093 RID: 147
		// (get) Token: 0x060002E2 RID: 738 RVA: 0x0000EAE5 File Offset: 0x0000CCE5
		// (set) Token: 0x060002E3 RID: 739 RVA: 0x0000EAED File Offset: 0x0000CCED
		[DefaultValue(SimpleFormSetupType.None)]
		public SimpleFormSetupType CurrentSimpleFormSetupType { get; set; }

		// Token: 0x17000094 RID: 148
		// (get) Token: 0x060002E4 RID: 740 RVA: 0x0000EAF6 File Offset: 0x0000CCF6
		public FrameworkElement MainForm
		{
			get
			{
				return this.fromContent;
			}
		}

		// Token: 0x060002E5 RID: 741 RVA: 0x0000EB00 File Offset: 0x0000CD00
		public ManagedForm()
		{
			this.InitializeComponent();
			base.Focusable = true;
			EventAggregatorManager.Global.GetEvent<SearchKeywordEvent>().Subscribe(new Action<SearchKeywordEventArgs>(this.Subscribe_SearchKeywordEvent));
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Subscribe(new Action<SearchResultInfo>(this.Subscribe_SearchResultInfoSelected));
			base.Loaded += this.ManagedForm_Loaded;
			base.IsVisibleChanged += this.ManagedForm_IsVisibleChanged;
			base.AddHandler(UIElement.PreviewMouseDownEvent, new MouseButtonEventHandler(this.UserControl_MouseDown), true);
			base.DataContextChanged += this.ManagedForm_DataContextChanged;
			this.InitForm();
			this.IsSimpleForm = TzpManager.Current.IsSimpleForm;
			this.BindingCommands();
			if (File.Exists("C:\\TT\\debug"))
			{
				this.output4fd.Visibility = Visibility.Visible;
			}
			ManagedForm.current = this;
		}

		// Token: 0x060002E6 RID: 742 RVA: 0x0000EBE7 File Offset: 0x0000CDE7
		private void ManagedForm_PreviewMouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			this.DoubleClickTimer.Start();
		}

		// Token: 0x060002E7 RID: 743 RVA: 0x0000EBFB File Offset: 0x0000CDFB
		private void OnDoubleClick(object sender, EventArgs e)
		{
			this.DoubleClickTimer.Stop();
			this.FindFieldForCodeEditor();
		}

		// Token: 0x060002E8 RID: 744 RVA: 0x0000EC10 File Offset: 0x0000CE10
		private void FindFieldForCodeEditor()
		{
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() != 1)
			{
				return;
			}
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (!xmlElement.IsFormField)
			{
				return;
			}
			string text = xmlElement.Name;
			string[] array = text.Split(new char[] { '.' });
			if (array.Length > 1)
			{
				text = array[1];
			}
			FieldArgs fieldArgs = new FieldArgs(text, this.Key);
			EventAggregatorManager.Global.GetEvent<SearchCodeFromFormEvent>().Publish(fieldArgs);
		}

		// Token: 0x060002E9 RID: 745 RVA: 0x0000EC9C File Offset: 0x0000CE9C
		private void UserControl_MouseDown(object sender, MouseButtonEventArgs e)
		{
			ManagedForm.current = this;
			if (!this.IsSimpleForm)
			{
				TextBoxHelper.ForceUpdateSource();
			}
			base.Focus();
			Keyboard.Focus(this);
			if (this.IsSimpleForm)
			{
				FrameworkElement frameworkElement = e.OriginalSource as FrameworkElement;
				if (frameworkElement != null)
				{
					XmlElement xmlElement = frameworkElement.DataContext as XmlElement;
					if (xmlElement != null)
					{
						switch (this.CurrentSimpleFormSetupType)
						{
						case SimpleFormSetupType.None:
							break;
						case SimpleFormSetupType.Hide:
							if (!xmlElement.IsHidden)
							{
								xmlElement.IsHidden = true;
								return;
							}
							break;
						case SimpleFormSetupType.Show:
							if (xmlElement.IsHidden)
							{
								xmlElement.IsHidden = false;
							}
							break;
						default:
							return;
						}
					}
				}
			}
		}

		// Token: 0x060002EA RID: 746 RVA: 0x0000ED2C File Offset: 0x0000CF2C
		private void BindingCommands()
		{
			if (!this.IsSimpleForm)
			{
				base.CommandBindings.Add(new CommandBinding(FormCommands.GroupIntoCommand, null, new CanExecuteRoutedEventHandler(this.CanGroupInfo)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.AddToContainerCommand, new ExecutedRoutedEventHandler(this.ExecutedAddToContainer), new CanExecuteRoutedEventHandler(this.CanExecutedAddToContainer)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.BreakLayoutCommand, new ExecutedRoutedEventHandler(this.ExecutedBreakLayout), new CanExecuteRoutedEventHandler(this.CanBreakLayout)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.ConvertToWidgetCommand, null, new CanExecuteRoutedEventHandler(this.CanExecuteConvertToWidget)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.ConvertWidgetCommand, new ExecutedRoutedEventHandler(this.ExecutedConvertToWidget), new CanExecuteRoutedEventHandler(this.CanExecuteConvertWidget)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.ConvertToContainerCommand, null, new CanExecuteRoutedEventHandler(this.CanExecuteConvertToContainer)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.ConvertContainerCommand, new ExecutedRoutedEventHandler(this.ExecutedConvertToContainer), new CanExecuteRoutedEventHandler(this.CanExecuteConvertContainer)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.InsertAfterWidgetsCommand, new ExecutedRoutedEventHandler(this.ExecutedInsertAfterWidgets), new CanExecuteRoutedEventHandler(this.CanExecutedInsertWidgets)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.InsertBeforeWidgetsCommand, new ExecutedRoutedEventHandler(this.ExecutedInsertBeforeWidgets), new CanExecuteRoutedEventHandler(this.CanExecutedInsertWidgets)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.InsertReferenceCommand, new ExecutedRoutedEventHandler(this.ExecutedInsertReference), new CanExecuteRoutedEventHandler(this.CanExecutedInsertWidgets)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.InsertMultiLanguageCommand, new ExecutedRoutedEventHandler(this.ExecutedInsertMultiLanguage), new CanExecuteRoutedEventHandler(this.CanExecutedInsertWidgets)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.InsertQueryCommand, new ExecutedRoutedEventHandler(this.ExecutedInsertQuery), new CanExecuteRoutedEventHandler(this.CanExecutedInsertQuery)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.DeleteTableOrTreeWidgetsCommand, new ExecutedRoutedEventHandler(this.ExecutedDeleteWidgets)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.AddPageBefoerCommand, new ExecutedRoutedEventHandler(this.ExecuteAddPageBefore), new CanExecuteRoutedEventHandler(this.CanAddPage)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.AddPageAfterCommand, new ExecutedRoutedEventHandler(this.ExecuteAddPageAfter), new CanExecuteRoutedEventHandler(this.CanAddPage)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.DeletePageCommand, new ExecutedRoutedEventHandler(this.ExecuteDeletePage), new CanExecuteRoutedEventHandler(this.CanDeletePage)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.LayoutCommand, null, new CanExecuteRoutedEventHandler(this.CanLayoutCommand)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.HBoxLayoutCommand, new ExecutedRoutedEventHandler(this.ExecuteHBoxLayout)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.VBoxLayoutCommand, new ExecutedRoutedEventHandler(this.ExecuteVBoxLayout)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.MoveToFirstCommand, new ExecutedRoutedEventHandler(this.ExecuteMoveToFirst), new CanExecuteRoutedEventHandler(this.CanMoveToPrevious)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.MoveToPreviousCommand, new ExecutedRoutedEventHandler(this.ExecuteMoveToPrevious), new CanExecuteRoutedEventHandler(this.CanMoveToPrevious)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.MoveToNextCommand, new ExecutedRoutedEventHandler(this.ExecuteMoveToNext), new CanExecuteRoutedEventHandler(this.CanMoveToNext)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.MoveToLastCommand, new ExecutedRoutedEventHandler(this.ExecuteMoveToLast), new CanExecuteRoutedEventHandler(this.CanMoveToNext)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.AlignmentWidgetsCommand, null, new CanExecuteRoutedEventHandler(this.CanAlignWidgets)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.AlignToCommand, new ExecutedRoutedEventHandler(this.ExecuteAlign)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.BindingSelectionCommand, new ExecutedRoutedEventHandler(this.ExecutedBindingSelection), new CanExecuteRoutedEventHandler(this.CanExecuteBindingSelection)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.UnbindingSelectionCommand, new ExecutedRoutedEventHandler(this.ExecutedUnbindingSelection), new CanExecuteRoutedEventHandler(this.CanExecuteUnbindingSelection)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.GeneralFunctionCommand, new ExecutedRoutedEventHandler(this.ExecutedGeneralFunction), new CanExecuteRoutedEventHandler(this.CanExecuteGeneralFunction)));
				base.CommandBindings.Add(new CommandBinding(FormCommands.GeneralValueFuncCommand, new ExecutedRoutedEventHandler(this.ExecutedGeneralValueFunc), new CanExecuteRoutedEventHandler(this.CanExecuteGeneralValueFunc)));
			}
			base.CommandBindings.Add(new CommandBinding(FormCommands.MoveCommand, new ExecutedRoutedEventHandler(this.ExecutedMove), new CanExecuteRoutedEventHandler(this.CanExecuteMove)));
			if (!this.IsSimpleForm)
			{
				base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Cut, new ExecutedRoutedEventHandler(this.ExecuteCut), new CanExecuteRoutedEventHandler(this.CanCut)));
				base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Copy, new ExecutedRoutedEventHandler(this.ExecuteCopy), new CanExecuteRoutedEventHandler(this.CanCopy)));
				base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Paste, new ExecutedRoutedEventHandler(this.ExecutePaste), new CanExecuteRoutedEventHandler(this.CanPaste)));
				base.CommandBindings.Add(new CommandBinding(ApplicationCommands.Delete, new ExecutedRoutedEventHandler(this.ExecuteDelete), new CanExecuteRoutedEventHandler(this.CanDelete)));
				base.CommandBindings.Add(new CommandBinding(TabIndexCommands.SetAsFirstCommand, new ExecutedRoutedEventHandler(this.ExecutedSetAsFirst), new CanExecuteRoutedEventHandler(this.CanExecuteSetIndex)));
				base.CommandBindings.Add(new CommandBinding(TabIndexCommands.SetAsNextcommand, new ExecutedRoutedEventHandler(this.ExecutedSetAsNext), new CanExecuteRoutedEventHandler(this.CanExecuteSetAsNext)));
				base.CommandBindings.Add(new CommandBinding(TabIndexCommands.SetAsCurrentCommand, new ExecutedRoutedEventHandler(this.ExecutedSetAsCurrent), new CanExecuteRoutedEventHandler(this.CanExecuteSetIndex)));
				base.CommandBindings.Add(new CommandBinding(TabIndexCommands.ShiftCurrentCommand, new ExecutedRoutedEventHandler(this.ExecutedShiftCurrent), new CanExecuteRoutedEventHandler(this.CanShiftCurrent)));
				base.CommandBindings.Add(new CommandBinding(TabIndexCommands.SwapSelectedCommand, new ExecutedRoutedEventHandler(this.ExecutedSwapSelected), new CanExecuteRoutedEventHandler(this.CanExecuteSwapSelected)));
				base.CommandBindings.Add(new CommandBinding(TabIndexCommands.SetAsNonTabableCommand, new ExecutedRoutedEventHandler(this.SetAsNonTabableCommandExecuted), new CanExecuteRoutedEventHandler(this.CanExecuteSetIndex)));
			}
		}

		// Token: 0x17000095 RID: 149
		// (get) Token: 0x060002EB RID: 747 RVA: 0x0000F418 File Offset: 0x0000D618
		public static ManagedForm Current
		{
			get
			{
				return ManagedForm.current;
			}
		}

		// Token: 0x060002EC RID: 748 RVA: 0x0000F420 File Offset: 0x0000D620
		public void CanPaste(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() != 1)
			{
				e.CanExecute = false;
				return;
			}
			IDataObject dataObject = Clipboard.GetDataObject();
			string fullName = typeof(DesignerClipboardData).FullName;
			if (!dataObject.GetDataPresent(typeof(DesignerClipboardData)))
			{
				e.CanExecute = false;
				return;
			}
			DesignerClipboardData designerClipboardData = dataObject.GetData(fullName) as DesignerClipboardData;
			if (designerClipboardData == null || designerClipboardData.Components.Count == 0)
			{
				e.CanExecute = false;
				return;
			}
			bool flag = false;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (FormDesignSetting.IsContainer(xmlElement.NodeName))
			{
				flag = designerClipboardData.CanExecutePaste(xmlElement);
			}
			else if (xmlElement.Parent != null)
			{
				flag = designerClipboardData.CanExecutePaste(xmlElement.Parent);
			}
			e.CanExecute = flag;
		}

		// Token: 0x060002ED RID: 749 RVA: 0x0000F4FC File Offset: 0x0000D6FC
		public void ExecutePaste(object sender, ExecutedRoutedEventArgs e)
		{
			IDataObject dataObject = Clipboard.GetDataObject();
			DesignerClipboardData designerClipboardData = dataObject.GetData(typeof(DesignerClipboardData).FullName) as DesignerClipboardData;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (!FormDesignSetting.IsContainer(xmlElement.NodeName))
			{
				xmlElement = xmlElement.Parent;
			}
			try
			{
				designerClipboardData.ImportlocalItems(xmlElement.Key);
			}
			catch (Exception ex)
			{
				DSCLogger.Write(string.Format("{0} {1}", "Clipboard Error:", ex.Message), "");
				DesignerMessageBox.Show("多語言匯入失敗，請手動調整", "Error", MessageBoxButton.OK);
			}
			PasteComponentUndoRedoCommand pasteComponentUndoRedoCommand;
			if ((xmlElement != null && xmlElement.Type == ComponentType.Table) || xmlElement.Type == ComponentType.Tree)
			{
				pasteComponentUndoRedoCommand = new PasteComponentUndoRedoCommand(designerClipboardData, xmlElement, designerClipboardData.StartIndex + 1);
			}
			else
			{
				pasteComponentUndoRedoCommand = new PasteComponentUndoRedoCommand(designerClipboardData, xmlElement);
			}
			pasteComponentUndoRedoCommand.Execute();
		}

		// Token: 0x060002EE RID: 750 RVA: 0x0000F5E4 File Offset: 0x0000D7E4
		private void CanCut(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 0)
			{
				e.CanExecute = false;
				return;
			}
			if (ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent != null)
			{
				switch (ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					e.CanExecute = false;
					return;
				}
			}
			e.CanExecute = true;
		}

		// Token: 0x060002EF RID: 751 RVA: 0x0000F678 File Offset: 0x0000D878
		private void ExecuteCut(object sender, ExecutedRoutedEventArgs e)
		{
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				if (FormDesignSetting.IsContainer(xmlElement.NodeName))
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_NotSupportCutContainer") as string, Application.Current.FindResource("CE_Cut") as string, MessageBoxButton.OK);
					return;
				}
				if (xmlElement.IsCantDel)
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_NotSupportCutCantDel") as string, Application.Current.FindResource("CE_Cut") as string, MessageBoxButton.OK);
					return;
				}
			}
			CutComponentUndoRedoCommand cutComponentUndoRedoCommand = new CutComponentUndoRedoCommand(ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent, ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>());
			SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(cutComponentUndoRedoCommand);
		}

		// Token: 0x060002F0 RID: 752 RVA: 0x0000F794 File Offset: 0x0000D994
		public void CanCopy(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 0)
			{
				e.CanExecute = false;
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				if (FormDesignSetting.IsContainer(xmlElement.NodeName))
				{
					e.CanExecute = false;
					return;
				}
			}
			e.CanExecute = true;
		}

		// Token: 0x060002F1 RID: 753 RVA: 0x0000F828 File Offset: 0x0000DA28
		public void ExecuteCopy(object sender, ExecutedRoutedEventArgs e)
		{
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				if (FormDesignSetting.IsContainer(xmlElement.NodeName))
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_NotSupportCopyContainer") as string, Application.Current.FindResource("CE_Copy") as string, MessageBoxButton.OK);
					return;
				}
			}
			Clipboard.Clear();
			try
			{
				DesignerClipboardData designerClipboardData = new DesignerClipboardData(this.Key, (ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent == null) ? ComponentType.Unknown : ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent.Type);
				foreach (XmlElement xmlElement2 in ComponentHelper.Get(this.Key).SelectedObjects)
				{
					designerClipboardData.AppendSelectedItem(xmlElement2);
				}
				IDataObject dataObject = new DataObject();
				dataObject.SetData(typeof(DesignerClipboardData).FullName, designerClipboardData);
				Clipboard.SetDataObject(designerClipboardData, false);
			}
			catch (Exception ex)
			{
				DesignerMessageBox.Show("Oops! 剪貼簿出錯了!", "Error", MessageBoxButton.OK);
				DSCLogger.Write(ex);
			}
		}

		// Token: 0x060002F2 RID: 754 RVA: 0x0000F9A8 File Offset: 0x0000DBA8
		public void CanDelete(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = e.Source is ManagedForm;
			if (flag && e.Parameter != null)
			{
				flag = !e.Parameter.ToString().Equals("ContextMenuDelete");
			}
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 0)
			{
				e.CanExecute = false;
				return;
			}
			bool flag2 = true;
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				if (!flag2)
				{
					break;
				}
				if (xmlElement.NodeName == ComponentType.Form.ToString() || xmlElement.Parent.NodeName == ComponentType.Form.ToString())
				{
					if (flag)
					{
						DesignerMessageBox.Show(Application.Current.FindResource("Message_CantDeleteFormComponent") as string);
					}
					return;
				}
				if (!xmlElement.CanDelIncludeChildren)
				{
					if (flag)
					{
						string text = Application.Current.FindResource("Message_IncludeCantDelComponent") as string;
						DesignerMessageBox.Show(string.Format(text, xmlElement.Name));
					}
					flag2 = false;
				}
			}
			e.CanExecute = flag2;
		}

		// Token: 0x060002F3 RID: 755 RVA: 0x0000FAF8 File Offset: 0x0000DCF8
		public void ExecuteDelete(object sender, ExecutedRoutedEventArgs e)
		{
			XmlElement parent = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent;
			List<XmlElement> list = new List<XmlElement>();
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				list.Add(xmlElement);
				if (xmlElement.BindElement != null)
				{
					list.Add(xmlElement.BindElement);
					xmlElement.ClearBinding();
				}
			}
			DeleteComponentsUndoRedoCommand deleteComponentsUndoRedoCommand = new DeleteComponentsUndoRedoCommand(parent, list);
			deleteComponentsUndoRedoCommand.Execute();
		}

		// Token: 0x060002F4 RID: 756 RVA: 0x0000FC00 File Offset: 0x0000DE00
		private void CanExecuteMove(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 0 || !(e.Parameter is MoveDirection))
			{
				e.CanExecute = false;
				return;
			}
			MoveDirection direction = (MoveDirection)e.Parameter;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (xmlElement.Parent != null && (xmlElement.Parent.NodeName == ComponentType.VBox.ToString() || xmlElement.Parent.NodeName == ComponentType.HBox.ToString()))
			{
				return;
			}
			if (this.CantMovableInSimpleForm())
			{
				e.CanExecute = false;
				return;
			}
			List<XmlElement> list = ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>();
			list.Sort(delegate(XmlElement x, XmlElement y)
			{
				if (x == null)
				{
					if (y == null)
					{
						return 0;
					}
					return -1;
				}
				else
				{
					if (y == null)
					{
						return 0;
					}
					switch (direction)
					{
					case MoveDirection.Up:
					case MoveDirection.Down:
						return x.GridY - y.GridY;
					case MoveDirection.Left:
					case MoveDirection.Right:
						return x.GridX - y.GridX;
					default:
						return 0;
					}
				}
			});
			bool flag = false;
			foreach (XmlElement xmlElement2 in list)
			{
				if (flag)
				{
					break;
				}
				switch (direction)
				{
				case MoveDirection.Up:
				{
					int gridY = xmlElement2.GridY;
					flag = xmlElement2.MinGridY > xmlElement2.GridY - 1;
					break;
				}
				case MoveDirection.Left:
				{
					int gridX = xmlElement2.GridX;
					flag = xmlElement2.MinGridX > xmlElement2.GridX - 1;
					break;
				}
				}
			}
			e.CanExecute = !flag;
		}

		// Token: 0x060002F5 RID: 757 RVA: 0x0000FD88 File Offset: 0x0000DF88
		public bool CantMovableInSimpleForm()
		{
			bool flag = false;
			if (this.IsSimpleForm)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				if (xmlElement != null)
				{
					if (xmlElement.Parent != null && (xmlElement.Parent.NodeName == ComponentType.Table.ToString() || xmlElement.Parent.NodeName == ComponentType.ScrollGrid.ToString()))
					{
						flag = true;
					}
					if (xmlElement.NodeName == ComponentType.VBox.ToString() || xmlElement.NodeName == ComponentType.HBox.ToString() || xmlElement.NodeName == ComponentType.Grid.ToString() || xmlElement.NodeName == ComponentType.Folder.ToString() || xmlElement.NodeName == ComponentType.Group.ToString() || xmlElement.NodeName == ComponentType.Page.ToString() || xmlElement.NodeName == ComponentType.ScrollGrid.ToString() || xmlElement.NodeName == ComponentType.Table.ToString() || xmlElement.NodeName == ComponentType.Tree.ToString() || xmlElement.NodeName == ComponentType.Unknown.ToString())
					{
						flag = true;
					}
				}
			}
			return flag;
		}

		// Token: 0x060002F6 RID: 758 RVA: 0x0000FF08 File Offset: 0x0000E108
		public void SimpleFormHideAll(bool val)
		{
			SpecificationInfo specificationInfo = base.DataContext as SpecificationInfo;
			if (specificationInfo != null)
			{
				XmlElement formNode = specificationInfo.FormNode;
				formNode.BatchSetHidden(val);
			}
		}

		// Token: 0x060002F7 RID: 759 RVA: 0x0000FF34 File Offset: 0x0000E134
		private void ExecutedMove(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			MoveDirection moveDirection = (MoveDirection)e.Parameter;
			MoveComponentsUndoRedoCommand moveComponentsUndoRedoCommand = new MoveComponentsUndoRedoCommand(ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>(), moveDirection, 1);
			moveComponentsUndoRedoCommand.Execute();
		}

		// Token: 0x060002F8 RID: 760 RVA: 0x0000FF77 File Offset: 0x0000E177
		private void CanGroupInfo(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = true;
		}

		// Token: 0x060002F9 RID: 761 RVA: 0x0000FF88 File Offset: 0x0000E188
		private void CanExecutedAddToContainer(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (!(e.Parameter is ComponentType))
			{
				e.CanExecute = false;
				return;
			}
			ComponentType componentType = (ComponentType)e.Parameter;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 0)
			{
				e.CanExecute = false;
				return;
			}
			XmlElement parent = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent;
			if (parent == null)
			{
				e.CanExecute = false;
				return;
			}
			switch (componentType)
			{
			case ComponentType.Grid:
			case ComponentType.Group:
				if (!ComponentFactory.AcceptMimes(parent, componentType))
				{
					e.CanExecute = false;
					return;
				}
				break;
			case ComponentType.Page:
				if (parent.Type != ComponentType.Page && !ComponentFactory.AcceptMimes(ComponentType.Folder.ToString(), componentType.ToString()))
				{
					e.CanExecute = false;
					return;
				}
				break;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				if (!ComponentFactory.AcceptMimes(componentType, xmlElement))
				{
					e.CanExecute = false;
					return;
				}
			}
			e.CanExecute = true;
		}

		// Token: 0x060002FA RID: 762 RVA: 0x000100C0 File Offset: 0x0000E2C0
		private void ExecutedAddToContainer(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement parent = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>().Parent;
			ComponentType componentType = (ComponentType)e.Parameter;
			AddToContainerUndoRedoCommand addToContainerUndoRedoCommand = new AddToContainerUndoRedoCommand(ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>(), parent, componentType);
			addToContainerUndoRedoCommand.Execute();
		}

		// Token: 0x060002FB RID: 763 RVA: 0x00010120 File Offset: 0x0000E320
		private void CanBreakLayout(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			IEnumerable<XmlElement> selectedObjects = ComponentHelper.Get(this.Key).SelectedObjects;
			if (1 == selectedObjects.Count<XmlElement>())
			{
				XmlElement xmlElement = selectedObjects.FirstOrDefault<XmlElement>();
				if (xmlElement == null)
				{
					return;
				}
				e.CanExecute = !xmlElement.IsCantDel && xmlElement.Parent != null && xmlElement.Parent.NodeName != ComponentType.Form.ToString() && xmlElement.Parent.NodeName != ComponentType.VBox.ToString() && xmlElement.Parent.NodeName != ComponentType.HBox.ToString() && (xmlElement.NodeName == ComponentType.VBox.ToString() || xmlElement.NodeName == ComponentType.HBox.ToString());
				if (xmlElement.Parent == null)
				{
					return;
				}
				using (IEnumerator<XmlElement> enumerator = xmlElement.Nodes.GetEnumerator())
				{
					while (enumerator.MoveNext())
					{
						XmlElement xmlElement2 = enumerator.Current;
						e.CanExecute &= ComponentFactory.AcceptMimes(xmlElement.Parent, xmlElement2.Type);
					}
					return;
				}
			}
			e.CanExecute = false;
		}

		// Token: 0x060002FC RID: 764 RVA: 0x0001026C File Offset: 0x0000E46C
		private void ExecutedBreakLayout(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			BreakLayoutUndoRedoCommand breakLayoutUndoRedoCommand = new BreakLayoutUndoRedoCommand(ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>());
			breakLayoutUndoRedoCommand.Execute();
		}

		// Token: 0x060002FD RID: 765 RVA: 0x000102A4 File Offset: 0x0000E4A4
		private void CanExecutedInsertWidgets(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			bool flag = false;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 1)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				if (xmlElement != null && xmlElement.Parent != null)
				{
					switch (xmlElement.Parent.Type)
					{
					case ComponentType.Table:
					case ComponentType.Tree:
						flag = true;
						break;
					}
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x060002FE RID: 766 RVA: 0x0001031C File Offset: 0x0000E51C
		private void ExecutedInsertBeforeWidgets(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			AddComponetsUndoRedoCommand addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(new List<XmlElement> { ComponentFactory.CreateEmptyComponent(this.Key, ComponentType.Edit, "") }, xmlElement.Parent, xmlElement.Index);
			addComponetsUndoRedoCommand.Execute();
		}

		// Token: 0x060002FF RID: 767 RVA: 0x00010380 File Offset: 0x0000E580
		private void ExecutedInsertAfterWidgets(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			AddComponetsUndoRedoCommand addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(new List<XmlElement> { ComponentFactory.CreateEmptyComponent(this.Key, ComponentType.Edit, "") }, xmlElement.Parent, xmlElement.Index + 1);
			addComponetsUndoRedoCommand.Execute();
		}

		// Token: 0x06000300 RID: 768 RVA: 0x000103E4 File Offset: 0x0000E5E4
		private void ExecutedInsertReference(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			AddComponetsUndoRedoCommand addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(new List<XmlElement> { ComponentFactory.CreateEmptyComponentForBody(this.Key, SpecNodeType.REFERENCE) }, xmlElement.Parent, xmlElement.Index + 1);
			addComponetsUndoRedoCommand.Execute();
		}

		// Token: 0x06000301 RID: 769 RVA: 0x00010444 File Offset: 0x0000E644
		private void ExecutedInsertMultiLanguage(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			AddComponetsUndoRedoCommand addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(new List<XmlElement> { ComponentFactory.CreateEmptyComponentForBody(this.Key, SpecNodeType.MULTILANG) }, xmlElement.Parent, xmlElement.Index + 1);
			addComponetsUndoRedoCommand.Execute();
		}

		// Token: 0x06000302 RID: 770 RVA: 0x000104A1 File Offset: 0x0000E6A1
		private void CanExecutedInsertQuery(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = "Q" == SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.GetCodeTemplate();
		}

		// Token: 0x06000303 RID: 771 RVA: 0x000104D4 File Offset: 0x0000E6D4
		private void ExecutedInsertQuery(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			AddComponetsUndoRedoCommand addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(new List<XmlElement> { ComponentFactory.CreateEmptyComponentForBody(this.Key, SpecNodeType.PROGREL) }, xmlElement.Parent, xmlElement.Index + 1);
			addComponetsUndoRedoCommand.Execute();
		}

		// Token: 0x06000304 RID: 772 RVA: 0x00010531 File Offset: 0x0000E731
		private void ExecutedDeleteWidgets(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ComponentHelper.Get(this.Key).DeleteSelection();
		}

		// Token: 0x06000305 RID: 773 RVA: 0x0001054C File Offset: 0x0000E74C
		private void CanExecuteConvertToWidget(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			bool flag = false;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 1)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				switch (this.GetSpecificationInfo().GetSpecType(xmlElement))
				{
				case SpecNodeType.PROGREL:
				case SpecNodeType.NONE:
				case SpecNodeType.REFERENCE:
				case SpecNodeType.MULTILANG:
					flag = false;
					break;
				default:
					flag = !FormDesignSetting.IsContainer(xmlElement.NodeName) && xmlElement.IsFormField;
					break;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000306 RID: 774 RVA: 0x000105DC File Offset: 0x0000E7DC
		private void CanExecuteConvertWidget(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			bool flag = false;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 1)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				if (e.Parameter is ComponentType)
				{
					ComponentType componentType = (ComponentType)e.Parameter;
					flag = componentType != xmlElement.Type;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000307 RID: 775 RVA: 0x00010650 File Offset: 0x0000E850
		private void ExecutedConvertToWidget(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ComponentType componentType = (ComponentType)e.Parameter;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			ConvertWidgetTypeUndoRedoCommand convertWidgetTypeUndoRedoCommand = new ConvertWidgetTypeUndoRedoCommand(xmlElement, componentType);
			convertWidgetTypeUndoRedoCommand.Execute();
		}

		// Token: 0x06000308 RID: 776 RVA: 0x00010694 File Offset: 0x0000E894
		private void CanExecuteConvertToContainer(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			bool flag = false;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 1)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				switch (xmlElement.Type)
				{
				case ComponentType.Grid:
				case ComponentType.Group:
					flag = true;
					break;
				default:
					flag = false;
					break;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x06000309 RID: 777 RVA: 0x00010700 File Offset: 0x0000E900
		private void CanExecuteConvertContainer(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			bool flag = false;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 1)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				if (e.Parameter is ComponentType)
				{
					ComponentType componentType = (ComponentType)e.Parameter;
					flag = componentType != xmlElement.Type;
				}
			}
			e.CanExecute = flag;
		}

		// Token: 0x0600030A RID: 778 RVA: 0x00010774 File Offset: 0x0000E974
		private void ExecutedConvertToContainer(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ComponentType componentType = (ComponentType)e.Parameter;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			ConvertContainerTypeUndoRedoCommand convertContainerTypeUndoRedoCommand = new ConvertContainerTypeUndoRedoCommand(xmlElement, componentType);
			convertContainerTypeUndoRedoCommand.Execute();
		}

		// Token: 0x0600030B RID: 779 RVA: 0x000107B8 File Offset: 0x0000E9B8
		private void CanMoveToPrevious(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = this.canMoveInBox();
			if (e.CanExecute)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				e.CanExecute = xmlElement.Index > 0;
			}
		}

		// Token: 0x0600030C RID: 780 RVA: 0x00010808 File Offset: 0x0000EA08
		private void ExecuteMoveToFirst(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			ChangeChildIndexUndoRedoCommand changeChildIndexUndoRedoCommand = new ChangeChildIndexUndoRedoCommand(xmlElement, 0);
			SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(changeChildIndexUndoRedoCommand);
		}

		// Token: 0x0600030D RID: 781 RVA: 0x00010850 File Offset: 0x0000EA50
		private void ExecuteMoveToPrevious(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			int num = xmlElement.Index;
			num = ((num > 0) ? (num - 1) : 0);
			ChangeChildIndexUndoRedoCommand changeChildIndexUndoRedoCommand = new ChangeChildIndexUndoRedoCommand(xmlElement, num);
			SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(changeChildIndexUndoRedoCommand);
		}

		// Token: 0x0600030E RID: 782 RVA: 0x000108AC File Offset: 0x0000EAAC
		private void CanMoveToNext(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			e.CanExecute = this.canMoveInBox();
			if (e.CanExecute)
			{
				XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
				e.CanExecute = xmlElement.Index < xmlElement.Parent.Nodes.Count - 1;
			}
		}

		// Token: 0x0600030F RID: 783 RVA: 0x0001090C File Offset: 0x0000EB0C
		private void ExecuteMoveToNext(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			ChangeChildIndexUndoRedoCommand changeChildIndexUndoRedoCommand = new ChangeChildIndexUndoRedoCommand(xmlElement, xmlElement.Index + 1);
			SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(changeChildIndexUndoRedoCommand);
		}

		// Token: 0x06000310 RID: 784 RVA: 0x0001095C File Offset: 0x0000EB5C
		private void ExecuteMoveToLast(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			ChangeChildIndexUndoRedoCommand changeChildIndexUndoRedoCommand = new ChangeChildIndexUndoRedoCommand(xmlElement, xmlElement.Parent.Nodes.Count - 1);
			SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(changeChildIndexUndoRedoCommand);
		}

		// Token: 0x06000311 RID: 785 RVA: 0x000109B8 File Offset: 0x0000EBB8
		private bool canMoveInBox()
		{
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() != 1)
			{
				return false;
			}
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (xmlElement.Parent == null)
			{
				return false;
			}
			ComponentType type = xmlElement.Parent.Type;
			if (type != ComponentType.Folder)
			{
				switch (type)
				{
				case ComponentType.HBox:
				case ComponentType.Table:
				case ComponentType.Tree:
				case ComponentType.VBox:
					break;
				default:
					return false;
				}
			}
			return xmlElement.Parent.Nodes.Count<XmlElement>() != 1;
		}

		// Token: 0x06000312 RID: 786 RVA: 0x00010A4C File Offset: 0x0000EC4C
		private void CanLayoutCommand(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() == 0)
			{
				e.CanExecute = false;
				return;
			}
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (xmlElement.Parent == null || xmlElement.Parent.Type == ComponentType.Form)
			{
				e.CanExecute = false;
				return;
			}
			if (!ComponentFactory.AcceptMimes(ComponentType.HBox.ToString(), xmlElement.NodeName))
			{
				e.CanExecute = false;
				return;
			}
			if (!ComponentFactory.AcceptMimes(xmlElement.Parent.NodeName, ComponentType.HBox.ToString()))
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x06000313 RID: 787 RVA: 0x00010B00 File Offset: 0x0000ED00
		private void ExecuteHBoxLayout(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			XmlElement parent = xmlElement.Parent;
			AddToContainerUndoRedoCommand addToContainerUndoRedoCommand = new AddToContainerUndoRedoCommand(ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>(), parent, ComponentType.HBox);
			addToContainerUndoRedoCommand.Execute();
		}

		// Token: 0x06000314 RID: 788 RVA: 0x00010B54 File Offset: 0x0000ED54
		private void ExecuteVBoxLayout(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			XmlElement parent = xmlElement.Parent;
			AddToContainerUndoRedoCommand addToContainerUndoRedoCommand = new AddToContainerUndoRedoCommand(ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>(), parent, ComponentType.VBox);
			addToContainerUndoRedoCommand.Execute();
		}

		// Token: 0x06000315 RID: 789 RVA: 0x00010BAC File Offset: 0x0000EDAC
		private void CanAlignWidgets(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() <= 1)
			{
				e.CanExecute = false;
				return;
			}
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (xmlElement.Parent == null)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x06000316 RID: 790 RVA: 0x00010C08 File Offset: 0x0000EE08
		private void ExecuteAlign(object sender, ExecutedRoutedEventArgs e)
		{
			AlignOptions alignOptions = (AlignOptions)e.Parameter;
			AlignUndoRedoCommand alignUndoRedoCommand = new AlignUndoRedoCommand(ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>(), alignOptions);
			SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(alignUndoRedoCommand);
		}

		// Token: 0x06000317 RID: 791 RVA: 0x00010C54 File Offset: 0x0000EE54
		private void CanAddPage(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() != 1)
			{
				e.CanExecute = false;
				return;
			}
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (xmlElement.Type != ComponentType.Page)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x06000318 RID: 792 RVA: 0x00010CB8 File Offset: 0x0000EEB8
		private void ExecuteAddPageBefore(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			AddComponetsUndoRedoCommand addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(new List<XmlElement> { ComponentFactory.CreateEmptyComponent(xmlElement.Key, ComponentType.Page, "") }, xmlElement.Parent, xmlElement.Index);
			addComponetsUndoRedoCommand.Execute();
		}

		// Token: 0x06000319 RID: 793 RVA: 0x00010D18 File Offset: 0x0000EF18
		private void ExecuteAddPageAfter(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			AddComponetsUndoRedoCommand addComponetsUndoRedoCommand = new AddComponetsUndoRedoCommand(new List<XmlElement> { ComponentFactory.CreateEmptyComponent(xmlElement.Key, ComponentType.Page, "") }, xmlElement.Parent, xmlElement.Index + 1);
			addComponetsUndoRedoCommand.Execute();
		}

		// Token: 0x0600031A RID: 794 RVA: 0x00010D7C File Offset: 0x0000EF7C
		private void CanDeletePage(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() != 1)
			{
				e.CanExecute = false;
				return;
			}
			XmlElement xmlElement = ComponentHelper.Get(this.Key).SelectedObjects.FirstOrDefault<XmlElement>();
			if (xmlElement.Type != ComponentType.Page)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = xmlElement.CanDelIncludeChildren;
		}

		// Token: 0x0600031B RID: 795 RVA: 0x00010DE3 File Offset: 0x0000EFE3
		private void ExecuteDeletePage(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			ComponentHelper.Get(this.Key).DeleteSelection();
		}

		// Token: 0x0600031C RID: 796 RVA: 0x00010DFC File Offset: 0x0000EFFC
		private void CanExecuteSetIndex(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService componentTabIndexService = ComponentTabIndexService.Get(xmlElement.Key);
			if (componentTabIndexService != null && componentTabIndexService.Contains(xmlElement))
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x0600031D RID: 797 RVA: 0x00010E44 File Offset: 0x0000F044
		private void ExecutedSetAsFirst(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService.Get(xmlElement.Key).SetAsFirst(xmlElement);
		}

		// Token: 0x0600031E RID: 798 RVA: 0x00010E78 File Offset: 0x0000F078
		private void CanExecuteSetAsNext(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService componentTabIndexService = ComponentTabIndexService.Get(xmlElement.Key);
			e.CanExecute = componentTabIndexService != null && componentTabIndexService.CurrentIndex != 0;
		}

		// Token: 0x0600031F RID: 799 RVA: 0x00010EBC File Offset: 0x0000F0BC
		private void ExecutedSetAsNext(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService.Get(xmlElement.Key).SetAsNext(xmlElement);
		}

		// Token: 0x06000320 RID: 800 RVA: 0x00010EF0 File Offset: 0x0000F0F0
		private void ExecutedSetAsCurrent(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService.Get(xmlElement.Key).SetAsCurrent(xmlElement);
		}

		// Token: 0x06000321 RID: 801 RVA: 0x00010F24 File Offset: 0x0000F124
		private void CanShiftCurrent(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService componentTabIndexService = ComponentTabIndexService.Get(xmlElement.Key);
			if (componentTabIndexService != null && componentTabIndexService.Contains(xmlElement))
			{
				e.CanExecute = xmlElement.TabIndex < ComponentTabIndexService.Get(xmlElement.Key).CurrentIndex;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x06000322 RID: 802 RVA: 0x00010F84 File Offset: 0x0000F184
		private void ExecutedShiftCurrent(object sender, ExecutedRoutedEventArgs e)
		{
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService.Get(xmlElement.Key).ShiftCurrent(xmlElement);
		}

		// Token: 0x06000323 RID: 803 RVA: 0x00010FB0 File Offset: 0x0000F1B0
		private void CanExecuteSwapSelected(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService componentTabIndexService = ComponentTabIndexService.Get(xmlElement.Key);
			if (componentTabIndexService != null)
			{
				e.CanExecute = componentTabIndexService.CurrentIndex != 0;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x06000324 RID: 804 RVA: 0x00010FFC File Offset: 0x0000F1FC
		private void ExecutedSwapSelected(object sender, ExecutedRoutedEventArgs e)
		{
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService.Get(xmlElement.Key).SwapSelected(xmlElement);
		}

		// Token: 0x06000325 RID: 805 RVA: 0x00011028 File Offset: 0x0000F228
		public void SetAsNonTabableCommandExecuted(object sender, ExecutedRoutedEventArgs e)
		{
			XmlElement xmlElement = e.Parameter as XmlElement;
			ComponentTabIndexService.Get(xmlElement.Key).SetAsNonTabable(xmlElement);
		}

		// Token: 0x06000326 RID: 806 RVA: 0x00011052 File Offset: 0x0000F252
		private SpecificationInfo GetSpecificationInfo()
		{
			if (!(null == this.Key))
			{
				return SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
			}
			return null;
		}

		// Token: 0x06000327 RID: 807 RVA: 0x0001107C File Offset: 0x0000F27C
		private void ManagedForm_DataContextChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			base.DataContextChanged -= this.ManagedForm_DataContextChanged;
			SpecificationInfo specificationInfo = e.NewValue as SpecificationInfo;
			if (specificationInfo != null)
			{
				this.Key = specificationInfo.Key;
				Interaction.GetBehaviors(this).Add(new KeyBehavior(this.Key));
				Interaction.GetBehaviors(this).Add(new DragSelectionBehavior());
				Interaction.GetBehaviors(this).Add(new DragDropBehavior(this.Key));
				this.notBookingWatermark.Visibility = (SettingManager.Get().GetTzpManger(this.Key).Booking ? Visibility.Collapsed : Visibility.Visible);
				base.PreviewMouseDoubleClick += this.ManagedForm_PreviewMouseDoubleClick;
				this.DoubleClickTimer = new DispatcherTimer(TimeSpan.FromMilliseconds(200.0), DispatcherPriority.Background, new EventHandler(this.OnDoubleClick), Dispatcher.CurrentDispatcher);
				ComponentTabIndexService.Get(this.Key).Register(specificationInfo.FormNode);
			}
		}

		// Token: 0x06000328 RID: 808 RVA: 0x0001116F File Offset: 0x0000F36F
		private void ManagedForm_IsVisibleChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			if (!(bool)e.NewValue && this.zoomer != null && this.zoomer.IsVisible)
			{
				this.zoomer.Close();
			}
		}

		// Token: 0x06000329 RID: 809 RVA: 0x000111A0 File Offset: 0x0000F3A0
		private void ManagedForm_Loaded(object sender, RoutedEventArgs e)
		{
			ManagedForm.current = this;
			ScreenRecordViewer.This.ProgramKey = ManagedForm.current.Key;
			TzpManager.Current = SettingManager.Get().GetTzpManger(ManagedForm.current.Key);
			this.MoveFocus(new TraversalRequest(FocusNavigationDirection.Next));
		}

		// Token: 0x0600032A RID: 810 RVA: 0x000111F0 File Offset: 0x0000F3F0
		private void Zoom_Click(object sender, RoutedEventArgs e)
		{
			if (this.zoomer == null)
			{
				this.zoomer = new FormZoomer(this.scroll);
				this.zoomer.Topmost = true;
			}
			this.zoomer.Show();
			this.zoomer.Closed += this.zoomer_Closed;
		}

		// Token: 0x0600032B RID: 811 RVA: 0x00011244 File Offset: 0x0000F444
		private void zoomer_Closed(object sender, EventArgs e)
		{
			this.zoomer.Closed -= this.zoomer_Closed;
			this.zoomer = null;
		}

		// Token: 0x0600032C RID: 812 RVA: 0x00011264 File Offset: 0x0000F464
		public void Subscribe_SearchKeywordEvent(SearchKeywordEventArgs searchArgs)
		{
			if (searchArgs.ProgramKey == null || searchArgs.ProgramKey == this.Key)
			{
				this.FindChildName(searchArgs);
			}
		}

		// Token: 0x0600032D RID: 813 RVA: 0x00011290 File Offset: 0x0000F490
		private void FindChildName(SearchKeywordEventArgs searchArgs)
		{
			SpecificationInfo specificationInfo = this.GetSpecificationInfo();
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair in specificationInfo.FormSpeDictionary)
			{
				if (keyValuePair.Key.Contains(searchArgs.Keyword))
				{
					SearchResultInfo searchResultInfo = new SearchResultInfo();
					searchResultInfo.Keyword = searchArgs.Keyword;
					searchResultInfo.Key = keyValuePair.Key;
					searchResultInfo.ProgramKey = specificationInfo.Key;
					searchResultInfo.SourceType = specificationInfo.Key.PackType;
					searchResultInfo.Match = string.Format("{0}:{1}", keyValuePair.Key, keyValuePair.Value.GeneroComponent.LocalString);
					EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo);
				}
				if (searchArgs.IsSearchSpec && keyValuePair.Value.SpecNode != null && keyValuePair.Value.SpecNode.CDATA.Contains(searchArgs.Keyword))
				{
					SearchResultInfo searchResultInfo2 = new SearchResultInfo();
					searchResultInfo2.Keyword = searchArgs.Keyword;
					searchResultInfo2.Key = keyValuePair.Key;
					searchResultInfo2.ProgramKey = specificationInfo.Key;
					searchResultInfo2.SourceType = specificationInfo.Key.PackType;
					searchResultInfo2.Match = string.Format("{0}:{1}{2}{3}", new object[]
					{
						keyValuePair.Key,
						keyValuePair.Value.GeneroComponent.LocalString,
						Environment.NewLine,
						keyValuePair.Value.SpecNode.CDATA
					});
					EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo2);
				}
			}
			if (searchArgs.IsSearchSpec && specificationInfo.ProgramDBSpec != null && specificationInfo.ProgramDBSpec.CDATA.Contains(searchArgs.Keyword))
			{
				SearchResultInfo searchResultInfo3 = new SearchResultInfo();
				searchResultInfo3.Keyword = searchArgs.Keyword;
				searchResultInfo3.Key = base.FindResource("specEditor_DbAllSpec") as string;
				searchResultInfo3.ProgramKey = specificationInfo.Key;
				searchResultInfo3.SourceType = specificationInfo.Key.PackType;
				searchResultInfo3.Match = string.Format("{0}:{1}{2}", searchResultInfo3.Key, Environment.NewLine, specificationInfo.ProgramDBSpec.CDATA);
				EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo3);
			}
			if (searchArgs.IsSearchSpec && specificationInfo.ProgramMISpec != null && specificationInfo.ProgramMISpec.CDATA.Contains(searchArgs.Keyword))
			{
				SearchResultInfo searchResultInfo4 = new SearchResultInfo();
				searchResultInfo4.Keyword = searchArgs.Keyword;
				searchResultInfo4.Key = base.FindResource("specEditor_MiAllSpec") as string;
				searchResultInfo4.ProgramKey = specificationInfo.Key;
				searchResultInfo4.SourceType = specificationInfo.Key.PackType;
				searchResultInfo4.Match = string.Format("{0}:{1}{2}", searchResultInfo4.Key, Environment.NewLine, specificationInfo.ProgramMISpec.CDATA);
				EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo4);
			}
			if (searchArgs.IsSearchSpec && specificationInfo.ProgramDISpec != null && specificationInfo.ProgramDISpec.CDATA.Contains(searchArgs.Keyword))
			{
				SearchResultInfo searchResultInfo5 = new SearchResultInfo();
				searchResultInfo5.Keyword = searchArgs.Keyword;
				searchResultInfo5.Key = base.FindResource("specEditor_DiAllSpec") as string;
				searchResultInfo5.ProgramKey = specificationInfo.Key;
				searchResultInfo5.SourceType = specificationInfo.Key.PackType;
				searchResultInfo5.Match = string.Format("{0}:{1}{2}", searchResultInfo5.Key, Environment.NewLine, specificationInfo.ProgramDISpec.CDATA);
				EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo5);
			}
			if (searchArgs.IsSearchSpec && specificationInfo.ProgramSpec != null && specificationInfo.ProgramSpec.CDATA.Contains(searchArgs.Keyword))
			{
				SearchResultInfo searchResultInfo6 = new SearchResultInfo();
				searchResultInfo6.Keyword = searchArgs.Keyword;
				searchResultInfo6.Key = base.FindResource("specEditor_AllSpec") as string;
				searchResultInfo6.ProgramKey = specificationInfo.Key;
				searchResultInfo6.SourceType = specificationInfo.Key.PackType;
				searchResultInfo6.Match = string.Format("{0}:{1}{2}", searchResultInfo6.Key, Environment.NewLine, specificationInfo.ProgramSpec.CDATA);
				EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Publish(searchResultInfo6);
			}
		}

		// Token: 0x0600032E RID: 814 RVA: 0x00011730 File Offset: 0x0000F930
		public void Subscribe_SearchResultInfoSelected(SearchResultInfo resultInfo)
		{
			if (!this.Key.Equals(resultInfo.ProgramKey))
			{
				return;
			}
			if (resultInfo.SourceType != TzpType.Form)
			{
				return;
			}
			this.scrollIntoViewHeadled = false;
			SpecificationInfo specificationInfo = this.GetSpecificationInfo();
			FormSpecModel formSpecModel = specificationInfo.FindNodeByName(resultInfo.Key);
			if (formSpecModel == null)
			{
				return;
			}
			XmlElement xmlElement = formSpecModel.GeneroComponent;
			while ((xmlElement = xmlElement.Parent) != null)
			{
				if (xmlElement.Type == ComponentType.Page)
				{
					xmlElement.IsSelected = true;
				}
			}
			ComponentHelper.Get(this.Key).AddSelection(formSpecModel.GeneroComponent, false);
		}

		// Token: 0x0600032F RID: 815 RVA: 0x000117B4 File Offset: 0x0000F9B4
		private void InitForm()
		{
			Interaction.GetBehaviors(this.main_panel).Add(new MouseWheelScaleBehavior());
			base.CommandBindings.Add(new CommandBinding(ScrollBarCommands.ZoomInCommand, new ExecutedRoutedEventHandler(this.ExecutedZoomIn), new CanExecuteRoutedEventHandler(this.CanExecuteZoomIn)));
			base.CommandBindings.Add(new CommandBinding(ScrollBarCommands.ZoomOutCommand, new ExecutedRoutedEventHandler(this.ExecutedZoomOut), new CanExecuteRoutedEventHandler(this.CanExecuteZoomOut)));
			base.CommandBindings.Add(new CommandBinding(ScrollBarCommands.ResetCommand, new ExecutedRoutedEventHandler(this.ExecutedReset)));
		}

		// Token: 0x06000330 RID: 816 RVA: 0x00011854 File Offset: 0x0000FA54
		private void CanExecuteZoomIn(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			ScaleTransform scaleTransform = this.main_panel.RenderTransform as ScaleTransform;
			e.CanExecute = scaleTransform.ScaleX > 0.5;
		}

		// Token: 0x06000331 RID: 817 RVA: 0x00011890 File Offset: 0x0000FA90
		private void ExecutedZoomIn(object sender, ExecutedRoutedEventArgs e)
		{
			ScaleTransform scaleTransform = (this.scroll.Content as FrameworkElement).RenderTransform as ScaleTransform;
			scaleTransform.ScaleX -= 0.1;
			scaleTransform.ScaleY -= 0.1;
		}

		// Token: 0x06000332 RID: 818 RVA: 0x000118E4 File Offset: 0x0000FAE4
		private void CanExecuteZoomOut(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			ScaleTransform scaleTransform = this.main_panel.RenderTransform as ScaleTransform;
			e.CanExecute = scaleTransform.ScaleX < 1.5;
		}

		// Token: 0x06000333 RID: 819 RVA: 0x00011920 File Offset: 0x0000FB20
		private void ExecutedZoomOut(object sender, ExecutedRoutedEventArgs e)
		{
			ScaleTransform scaleTransform = (this.scroll.Content as FrameworkElement).RenderTransform as ScaleTransform;
			scaleTransform.ScaleX += 0.1;
			scaleTransform.ScaleY += 0.1;
		}

		// Token: 0x06000334 RID: 820 RVA: 0x00011974 File Offset: 0x0000FB74
		private void ExecutedReset(object sender, ExecutedRoutedEventArgs e)
		{
			ScaleTransform scaleTransform = (this.scroll.Content as FrameworkElement).RenderTransform as ScaleTransform;
			scaleTransform.ScaleX = 1.0;
			scaleTransform.ScaleY = 1.0;
		}

		// Token: 0x06000335 RID: 821 RVA: 0x000119BC File Offset: 0x0000FBBC
		private void output4fd_Click(object sender, RoutedEventArgs e)
		{
			if (null == this.Key)
			{
				return;
			}
			SpecificationInfo specificationInfo = this.GetSpecificationInfo();
			specificationInfo.SaveToTSD();
			specificationInfo.SaveToForm();
			TestPropertyWindow.Show(specificationInfo.FormElement.ToString(), specificationInfo.TSDElement.ToString());
		}

		// Token: 0x06000336 RID: 822 RVA: 0x00011A08 File Offset: 0x0000FC08
		private void scroll_RequestBringIntoView(object sender, RequestBringIntoViewEventArgs e)
		{
			e.Handled = this.scrollIntoViewHeadled;
		}

		// Token: 0x06000337 RID: 823 RVA: 0x00011A18 File Offset: 0x0000FC18
		public void Dispose()
		{
			EventAggregatorManager.Global.GetEvent<SearchKeywordEvent>().Unsubscribe(new Action<SearchKeywordEventArgs>(this.Subscribe_SearchKeywordEvent));
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Unsubscribe(new Action<SearchResultInfo>(this.Subscribe_SearchResultInfoSelected));
			base.Loaded -= this.ManagedForm_Loaded;
			base.IsVisibleChanged -= this.ManagedForm_IsVisibleChanged;
			base.RemoveHandler(UIElement.PreviewMouseLeftButtonDownEvent, new MouseButtonEventHandler(this.UserControl_MouseDown));
			base.RemoveHandler(UIElement.PreviewMouseDownEvent, new MouseButtonEventHandler(this.UserControl_MouseDown));
			Interaction.GetBehaviors(this.main_panel).Clear();
			base.CommandBindings.Clear();
			if (this.zoomer != null)
			{
				this.zoomer.Closed -= this.zoomer_Closed;
				this.zoomer.Dispose();
			}
			if (this.Key != null)
			{
				ComponentTabIndexService.Get(this.Key).UnRegister();
			}
			ManagedForm.current = null;
		}

		// Token: 0x06000338 RID: 824 RVA: 0x00011B18 File Offset: 0x0000FD18
		private void CanExecuteBindingSelection(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() != 2)
			{
				e.CanExecute = false;
				return;
			}
			List<XmlElement> list = ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>();
			bool flag2;
			bool flag = (flag2 = false);
			foreach (XmlElement xmlElement in list)
			{
				if (xmlElement.Type == ComponentType.Label)
				{
					flag2 = true;
				}
				if (SpecificationInfo.isValidBindingType(xmlElement.Type))
				{
					flag = true;
				}
			}
			if (flag2 && flag)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x06000339 RID: 825 RVA: 0x00011BD4 File Offset: 0x0000FDD4
		private void ExecutedBindingSelection(object sender, ExecutedRoutedEventArgs e)
		{
			List<XmlElement> list = ComponentHelper.Get(this.Key).SelectedObjects.ToList<XmlElement>();
			bool flag = false;
			if (list[0].BindElement != null || list[1].BindElement != null)
			{
				if (MessageBox.Show(Application.Current.FindResource("Message_BindSelection") as string, Application.Current.FindResource("Message_Inform") as string, MessageBoxButton.YesNoCancel) == MessageBoxResult.Yes)
				{
					flag = true;
				}
			}
			else
			{
				flag = true;
			}
			if (flag)
			{
				list[0].ClearBinding();
				list[1].ClearBinding();
				list[0].BindElement = list[1];
				list[1].BindElement = list[0];
				this.GetSpecificationInfo().AddSpecBinding(list[0], list[1]);
			}
		}

		// Token: 0x0600033A RID: 826 RVA: 0x00011CA8 File Offset: 0x0000FEA8
		private void CanExecuteUnbindingSelection(object sender, CanExecuteRoutedEventArgs e)
		{
			bool flag = false;
			e.Handled = true;
			if (ComponentHelper.Get(this.Key).SelectedObjects.Count<XmlElement>() > 1)
			{
				e.CanExecute = false;
				return;
			}
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				if (xmlElement.BindElement != null)
				{
					flag = true;
				}
			}
			if (flag)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x0600033B RID: 827 RVA: 0x00011D40 File Offset: 0x0000FF40
		private void ExecutedUnbindingSelection(object sender, ExecutedRoutedEventArgs e)
		{
			foreach (XmlElement xmlElement in ComponentHelper.Get(this.Key).SelectedObjects)
			{
				xmlElement.ClearBinding();
			}
		}

		// Token: 0x0600033C RID: 828 RVA: 0x00011D98 File Offset: 0x0000FF98
		private bool IsGeneralFunctionComponent(XmlElement component)
		{
			return component.Type == ComponentType.ButtonEdit || component.Type == ComponentType.Edit || component.Type == ComponentType.DateEdit || component.Type == ComponentType.DateTimeEdit || component.Type == ComponentType.CheckBox || component.Type == ComponentType.ComboBox;
		}

		// Token: 0x0600033D RID: 829 RVA: 0x00011DD8 File Offset: 0x0000FFD8
		private void CanExecuteGeneralFunction(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			SpecificationInfo specificationInfo = base.DataContext as SpecificationInfo;
			IEnumerable<XmlElement> selectedObjects = ComponentHelper.Get(this.Key).SelectedObjects;
			if (selectedObjects.Count<XmlElement>() > 1)
			{
				e.CanExecute = false;
				return;
			}
			if (specificationInfo.Env == "s")
			{
				e.CanExecute = false;
				return;
			}
			if (this.IsGeneralFunctionComponent(selectedObjects.First<XmlElement>()))
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x0600033E RID: 830 RVA: 0x00011E51 File Offset: 0x00010051
		private void ExecutedGeneralFunction(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.GeneralFunction(this.Key, ComponentHelper.Get(this.Key).SelectedObjects.First<XmlElement>());
		}

		// Token: 0x0600033F RID: 831 RVA: 0x00011E74 File Offset: 0x00010074
		private void CanExecuteGeneralValueFunc(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			SpecificationInfo specificationInfo = base.DataContext as SpecificationInfo;
			IEnumerable<XmlElement> selectedObjects = ComponentHelper.Get(this.Key).SelectedObjects;
			if (selectedObjects.Count<XmlElement>() > 1)
			{
				e.CanExecute = false;
				return;
			}
			if (specificationInfo.Env == "s")
			{
				e.CanExecute = false;
				return;
			}
			if (this.IsGeneralFunctionComponent(selectedObjects.First<XmlElement>()))
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x06000340 RID: 832 RVA: 0x00011EED File Offset: 0x000100ED
		private void ExecutedGeneralValueFunc(object sender, ExecutedRoutedEventArgs e)
		{
			ConnectionManager.GeneralValueFunc(this.Key, ComponentHelper.Get(this.Key).SelectedObjects.First<XmlElement>());
		}

		// Token: 0x04000196 RID: 406
		private bool scrollIntoViewHeadled = true;

		// Token: 0x04000197 RID: 407
		private BackgroundWorker _worker;

		// Token: 0x04000198 RID: 408
		private FormZoomer zoomer;

		// Token: 0x04000199 RID: 409
		private DispatcherTimer DoubleClickTimer;

		// Token: 0x0400019A RID: 410
		private static ManagedForm current;

		// Token: 0x0400019B RID: 411
		private PackageKey Key;
	}
}
