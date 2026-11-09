using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.CodeEditWindow.Helper;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerPreference;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000029 RID: 41
	public partial class TreeView : UserControl, IDisposable
	{
		// Token: 0x1700002F RID: 47
		// (get) Token: 0x0600016D RID: 365 RVA: 0x0000D64A File Offset: 0x0000B84A
		// (set) Token: 0x0600016E RID: 366 RVA: 0x0000D652 File Offset: 0x0000B852
		private PackageKey ProgramName { get; set; }

		// Token: 0x17000030 RID: 48
		// (get) Token: 0x0600016F RID: 367 RVA: 0x0000D65B File Offset: 0x0000B85B
		public TreeItem SelectedItem
		{
			get
			{
				return this.treeView.SelectedItem as TreeItem;
			}
		}

		// Token: 0x06000170 RID: 368 RVA: 0x0000D670 File Offset: 0x0000B870
		public TreeView()
		{
			this.InitializeComponent();
			if (DesignerProperties.GetIsInDesignMode(this))
			{
				return;
			}
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.RefreshStructure, new ExecutedRoutedEventHandler(this.ExecutedRefresh)));
			base.CommandBindings.Add(new CommandBinding(CodeEditCommands.EnabledFunctionOrder, new ExecutedRoutedEventHandler(this.ExecutedAdjustFunctionSort), new CanExecuteRoutedEventHandler(this.CanAdjustFunctionSort)));
			this.treeView.KeyUp += this.treeView_KeyUp;
			this.treeView.PreviewMouseDoubleClick += this.treeView_MouseDoubleClick;
			Binding binding = new Binding("FontSize")
			{
				Source = PreferenceManager.Current.Settings,
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				Mode = BindingMode.OneWay
			};
			base.SetBinding(Control.FontSizeProperty, binding);
			Binding binding2 = new Binding("FontFamily")
			{
				Source = PreferenceManager.Current.Settings,
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				Mode = BindingMode.OneWay
			};
			base.SetBinding(Control.FontFamilyProperty, binding2);
			EventAggregatorManager.Global.GetEvent<TreeItemSort>().Subscribe(new Action<DiffTreeSortInfo>(this.OnTreeItemSort));
		}

		// Token: 0x06000171 RID: 369 RVA: 0x0000D79C File Offset: 0x0000B99C
		private void TreeViewItem_MouseRightButtonDown(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			TreeViewItem treeViewItem = sender as TreeViewItem;
			if (treeViewItem != null)
			{
				treeViewItem.IsSelected = true;
				treeViewItem.Focus();
			}
		}

		// Token: 0x06000172 RID: 370 RVA: 0x0000D7C8 File Offset: 0x0000B9C8
		public void Load(PackageKey programName)
		{
			this.ProgramName = programName;
			EventController.GetInstance().GetEvent<LoadedSettingEvent>().Subscribe(new Action<LoadInformation>(this.LoadedSetting));
		}

		// Token: 0x06000173 RID: 371 RVA: 0x0000D7ED File Offset: 0x0000B9ED
		protected override void OnMouseRightButtonDown(MouseButtonEventArgs e)
		{
			base.Focus();
			base.OnMouseRightButtonDown(e);
		}

		// Token: 0x06000174 RID: 372 RVA: 0x0000D800 File Offset: 0x0000BA00
		private void LoadedSetting(LoadInformation option)
		{
			if (this.ProgramName != option.key)
			{
				return;
			}
			if (option.IsDiff)
			{
				return;
			}
			EventController.GetInstance().GetEvent<LoadedSettingEvent>().Unsubscribe(new Action<LoadInformation>(this.LoadedSetting));
			base.DataContext = ResourceController.GetInstance().GetProgramInfo(option.key);
		}

		// Token: 0x06000175 RID: 373 RVA: 0x0000D85B File Offset: 0x0000BA5B
		private void ExecutedRefresh(object sender, ExecutedRoutedEventArgs e)
		{
			ResourceController.GetInstance().GetProgramInfo(this.ProgramName).RefreshTreeNodes();
			ResourceController.GetInstance().GetProgramInfo(this.ProgramName).VerifyAdjustFunctionSort = false;
		}

		// Token: 0x06000176 RID: 374 RVA: 0x0000D888 File Offset: 0x0000BA88
		private void OnTreeItemSort(DiffTreeSortInfo info)
		{
			TreeItem treeItem = null;
			TreeItem treeItem2 = null;
			foreach (TreeItem treeItem3 in ((TreeItem)this.treeView.Items[0]).Nodes)
			{
				if (treeItem3.IsFolder && treeItem3.Name == "FUNCTION")
				{
					ItemContainerGenerator itemContainerGenerator = this.treeView.ItemContainerGenerator;
					for (int i = 0; i <= treeItem3.Nodes.Count; i++)
					{
						if (i == info.SortIndex - 1)
						{
							treeItem = treeItem3.Nodes[i];
						}
						if (treeItem3.Nodes[i].Name == info.NodeName)
						{
							treeItem2 = treeItem3.Nodes[i];
						}
						if (treeItem != null && treeItem2 != null)
						{
							break;
						}
					}
					treeItem3.Remove(treeItem2);
					treeItem3.AddNode(treeItem2, info.SortIndex);
					EventAggregatorManager.Global.GetEvent<RefreshScreen>().Publish(info.ProgramKey);
					treeItem3.Nodes[info.SortIndex - 1].PublishSelectedEvent();
				}
			}
			ResourceController.GetInstance().GetProgramInfo(this.ProgramName).RefreshTreeNodes();
		}

		// Token: 0x06000177 RID: 375 RVA: 0x0000D9D4 File Offset: 0x0000BBD4
		private void CanAdjustFunctionSort(object sender, CanExecuteRoutedEventArgs e)
		{
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramName).VerifyAdjustFunctionSort)
			{
				e.CanExecute = false;
				return;
			}
			e.CanExecute = true;
		}

		// Token: 0x06000178 RID: 376 RVA: 0x0000D9FC File Offset: 0x0000BBFC
		private void ExecutedAdjustFunctionSort(object sender, ExecutedRoutedEventArgs e)
		{
			ResourceController.GetInstance().GetProgramInfo(this.ProgramName).VerifyAdjustFunctionSort = true;
		}

		// Token: 0x06000179 RID: 377 RVA: 0x0000DA14 File Offset: 0x0000BC14
		private void treeView_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			e.Handled = true;
			TreeItem treeItem = this.treeView.SelectedItem as TreeItem;
			if (treeItem == null)
			{
				return;
			}
			treeItem.PublishSelectedEvent();
		}

		// Token: 0x0600017A RID: 378 RVA: 0x0000DA44 File Offset: 0x0000BC44
		private void treeView_KeyUp(object sender, KeyEventArgs e)
		{
			TreeItem treeItem = this.treeView.SelectedItem as TreeItem;
			if (Key.Delete == e.Key && treeItem != null && CodeEditCommands.DeleteFunction.CanExecute(treeItem, null))
			{
				treeItem.DeleteCommand.Execute(string.Empty);
			}
		}

		// Token: 0x0600017B RID: 379 RVA: 0x0000DA90 File Offset: 0x0000BC90
		public void Dispose()
		{
			this.treeView.KeyUp -= this.treeView_KeyUp;
			this.treeView.PreviewMouseDoubleClick -= this.treeView_MouseDoubleClick;
			base.RemoveHandler(UIElement.PreviewMouseRightButtonDownEvent, new MouseButtonEventHandler(this.TreeViewItem_MouseRightButtonDown));
			BindingOperations.ClearAllBindings(this);
			base.CommandBindings.Clear();
		}

		// Token: 0x0600017C RID: 380 RVA: 0x0000DAF3 File Offset: 0x0000BCF3
		private void MenuItem_Click(object sender, RoutedEventArgs e)
		{
			Clipboard.SetData(DataFormats.UnicodeText, ResourceController.GetInstance().GetProgramInfo(this.ProgramName).AddPoint);
		}

		// Token: 0x0600017D RID: 381 RVA: 0x0000DB14 File Offset: 0x0000BD14
		private void ContextMenu_ContextMenuOpening(object sender, ContextMenuEventArgs e)
		{
			string text = string.Format("{0}\\debug", SettingManager.Get().CurrentSetting.Connection.Workspace);
			if (!File.Exists(text))
			{
				e.Handled = true;
			}
		}

		// Token: 0x0600017E RID: 382 RVA: 0x0000DB50 File Offset: 0x0000BD50
		private void TreeSearchBox_TextChanged(object sender, TextChangedEventArgs e)
		{
			foreach (TreeItem treeItem in ((TreeItem)this.treeView.Items[0]).Nodes)
			{
				if (!treeItem.IsFolder)
				{
					if (string.IsNullOrWhiteSpace(this.TreeSearchBox.Text))
					{
						treeItem.IsVisible = true;
					}
					else if (treeItem.DisplayName.Contains(this.TreeSearchBox.Text))
					{
						treeItem.IsVisible = true;
					}
					else if (!treeItem.DisplayName.Contains(this.TreeSearchBox.Text))
					{
						treeItem.IsVisible = false;
					}
				}
				else
				{
					foreach (TreeItem treeItem2 in treeItem.Nodes)
					{
						if (string.IsNullOrWhiteSpace(this.TreeSearchBox.Text))
						{
							treeItem2.IsVisible = true;
						}
						else if (treeItem2.DisplayName.Contains(this.TreeSearchBox.Text))
						{
							treeItem2.IsVisible = true;
						}
						else if (!treeItem2.DisplayName.Contains(this.TreeSearchBox.Text))
						{
							treeItem2.IsVisible = false;
						}
					}
				}
			}
		}

		// Token: 0x06000182 RID: 386 RVA: 0x0000DD64 File Offset: 0x0000BF64
		[DebuggerNonUserCode]
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		[EditorBrowsable(EditorBrowsableState.Never)]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			switch (connectionId)
			{
			case 1:
				((TextBlock)target).ContextMenuOpening += this.ContextMenu_ContextMenuOpening;
				return;
			case 2:
				((MenuItem)target).Click += this.MenuItem_Click;
				return;
			case 3:
			case 4:
				break;
			case 5:
			{
				EventSetter eventSetter = new EventSetter();
				eventSetter.Event = UIElement.MouseRightButtonDownEvent;
				eventSetter.Handler = new MouseButtonEventHandler(this.TreeViewItem_MouseRightButtonDown);
				((Style)target).Setters.Add(eventSetter);
				break;
			}
			default:
				return;
			}
		}
	}
}
