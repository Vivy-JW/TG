using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Media;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x0200001A RID: 26
	public partial class TableSchemaAssociationWindow : Window
	{
		// Token: 0x060000AC RID: 172 RVA: 0x00007810 File Offset: 0x00005A10
		public TableSchemaAssociationWindow(PackageKey programKey)
		{
			this._programKey = programKey;
			this.InitializeComponent();
			XElement source = SettingManager.Get().GetTzpManger(programKey).SpecificationInfo.AssociateTable.Source;
			TblModel tblModel = new TblModel(programKey.Program, null, source, programKey);
			this._tables = TableColumnHelper.GetTables();
			this.tablesTreeView.DataContext = tblModel;
			TableSchemaAssociationWindow._tablesTreeView = this.tablesTreeView;
			base.Closed += this.TableSchemaAssociationWindow_Closed;
			TableSchemaAssociationWindow._tablesProvider = base.FindResource("TablesProvider") as ObjectDataProvider;
			TableSchemaAssociationWindow._tablesProvider.ObjectInstance = new Tables(source);
			TableSchemaAssociationWindow._tablesProvider.MethodName = "GetTables";
		}

		// Token: 0x060000AD RID: 173 RVA: 0x000078F0 File Offset: 0x00005AF0
		private void TableSchemaAssociationWindow_Closed(object sender, EventArgs e)
		{
			XElement source = SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo.AssociateTable.Source;
			(from t in source.Elements("tbl")
				where t.Attribute("status").Value == ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE)
				select t).Remove<XElement>();
			BindingOperations.ClearAllBindings(this.tablesTreeView);
			this.grid = null;
			this.modifyBtn = null;
			this.applyBtn = null;
			this.addBtn = null;
			this.delBtn = null;
			this.readonlyTB = null;
			this.editableTB = null;
			this._programKey = null;
			this._tables = null;
		}

		// Token: 0x060000AE RID: 174 RVA: 0x000079A0 File Offset: 0x00005BA0
		private void ModifyButton_Click(object sender, RoutedEventArgs e)
		{
			if (this.modifyBtn != null)
			{
				this.modifyBtn.Visibility = Visibility.Collapsed;
			}
			if (this.applyBtn != null)
			{
				this.applyBtn.Visibility = Visibility.Visible;
			}
			if (this.editableTB != null)
			{
				this.editableTB.Visibility = Visibility.Visible;
				Keyboard.Focus(this.editableTB);
			}
			if (this.readonlyTB != null)
			{
				this.readonlyTB.Visibility = Visibility.Collapsed;
			}
			if (this.addBtn != null)
			{
				this.addBtn.Visibility = Visibility.Hidden;
			}
			if (this.delBtn != null)
			{
				this.delBtn.Visibility = Visibility.Hidden;
			}
		}

		// Token: 0x060000AF RID: 175 RVA: 0x00007A34 File Offset: 0x00005C34
		private void ApplyButton_Click(object sender, RoutedEventArgs e)
		{
			BindingExpression bindingExpression = this.editableTB.GetBindingExpression(ComboBox.TextProperty);
			bindingExpression.UpdateSource();
			this.modifyBtn.Visibility = Visibility.Visible;
			this.applyBtn.Visibility = Visibility.Collapsed;
			this.addBtn.Visibility = ((this._selectedTreeLevel == 2) ? Visibility.Collapsed : Visibility.Visible);
			this.delBtn.Visibility = Visibility.Visible;
			this.readonlyTB.Text = this.editableTB.Text;
			this.editableTB.Visibility = Visibility.Collapsed;
			this.readonlyTB.Visibility = Visibility.Visible;
		}

		// Token: 0x060000B0 RID: 176 RVA: 0x00007AC4 File Offset: 0x00005CC4
		private void tablesTreeView_Selected(object sender, RoutedEventArgs e)
		{
			TreeViewItem treeViewItem = e.OriginalSource as TreeViewItem;
			if (treeViewItem != null && treeViewItem.Header is TblModel)
			{
				this.currentItem = treeViewItem;
				this.grid = null;
				this.modifyBtn = null;
				this.applyBtn = null;
				this.addBtn = null;
				this.delBtn = null;
				this.readonlyTB = null;
				this.editableTB = null;
				ContentPresenter contentPresenter = (ContentPresenter)treeViewItem.Template.FindName("PART_Header", treeViewItem);
				if (contentPresenter == null)
				{
					TreeViewItem treeViewItem2 = TableSchemaAssociationWindow._tablesTreeView.ItemContainerGenerator.ContainerFromIndex(0) as TreeViewItem;
					if (treeViewItem2 != null)
					{
						treeViewItem2.IsSelected = true;
						treeViewItem2.IsSelected = false;
					}
					return;
				}
				if (contentPresenter != null)
				{
					this.grid = (Grid)VisualTreeHelper.GetChild(contentPresenter, 0);
					this.modifyBtn = this.grid.FindName("ModifyButton") as Button;
					FrameworkElement frameworkElement = this.modifyBtn.Parent as FrameworkElement;
					while (frameworkElement != null && frameworkElement.Name != "XXXXX")
					{
						if (frameworkElement.Name == "stackPanel")
						{
							this.applyBtn = frameworkElement.FindName("ApplyButton") as Button;
							this.modifyBtn = frameworkElement.FindName("ModifyButton") as Button;
							this.addBtn = frameworkElement.FindName("AddButton") as Button;
							this.delBtn = frameworkElement.FindName("DelButton") as Button;
						}
						frameworkElement = frameworkElement.Parent as FrameworkElement;
					}
					if (frameworkElement == null)
					{
						return;
					}
					this.readonlyTB = frameworkElement.FindName("ReadonlyTB") as TextBlock;
					this.editableTB = frameworkElement.FindName("EditableTB") as ComboBox;
				}
				this._selectedTreeLevel = this.FindTreeLevel(treeViewItem);
				TblModel tblModel = treeViewItem.Header as TblModel;
				this.modifyBtn.IsEnabled = tblModel != null && tblModel.IsEditable;
				this.applyBtn.Visibility = Visibility.Hidden;
				this.readonlyTB.Visibility = Visibility.Visible;
				this.editableTB.Visibility = Visibility.Collapsed;
				this.editableTB.Text = this.readonlyTB.Text;
			}
		}

		// Token: 0x060000B1 RID: 177 RVA: 0x00007CE0 File Offset: 0x00005EE0
		private void EditableTB_KeyDown(object sender, KeyEventArgs e)
		{
			if (e.Key == Key.Return)
			{
				BindingExpression bindingExpression = this.editableTB.GetBindingExpression(ComboBox.TextProperty);
				bindingExpression.UpdateSource();
				this.readonlyTB.Text = this.editableTB.Text;
				this.editableTB.Visibility = Visibility.Collapsed;
				this.readonlyTB.Visibility = Visibility.Visible;
				this.applyBtn.Visibility = Visibility.Collapsed;
				this.modifyBtn.Visibility = Visibility.Visible;
				this.addBtn.Visibility = ((this._selectedTreeLevel == 2) ? Visibility.Collapsed : Visibility.Visible);
				this.delBtn.Visibility = Visibility.Visible;
				this.initTemplateItems(this.currentItem);
				return;
			}
			if (e.Key == Key.Escape)
			{
				this.editableTB.GetBindingExpression(ComboBox.TextProperty);
				this.editableTB.Text = this.readonlyTB.Text;
				this.editableTB.Visibility = Visibility.Collapsed;
				this.readonlyTB.Visibility = Visibility.Visible;
				this.applyBtn.Visibility = Visibility.Collapsed;
				this.modifyBtn.Visibility = Visibility.Visible;
				this.addBtn.Visibility = ((this._selectedTreeLevel == 2) ? Visibility.Collapsed : Visibility.Visible);
				this.delBtn.Visibility = Visibility.Visible;
				this.initTemplateItems(this.currentItem);
			}
		}

		// Token: 0x060000B2 RID: 178 RVA: 0x00007E1B File Offset: 0x0000601B
		private void tablesTreeView_SelectedItemChanged(object sender, RoutedPropertyChangedEventArgs<object> e)
		{
			if (e.OriginalSource is TreeViewItem)
			{
				this.initTemplateItems(e.OriginalSource as TreeViewItem);
			}
		}

		// Token: 0x060000B3 RID: 179 RVA: 0x00007E3C File Offset: 0x0000603C
		private void initTemplateItems(TreeViewItem preTreeViewItem)
		{
			Button button = null;
			Button button2 = null;
			Button button3 = null;
			Button button4 = null;
			TextBlock textBlock = null;
			ComboBox comboBox = null;
			ContentPresenter contentPresenter = (ContentPresenter)preTreeViewItem.Template.FindName("PART_Header", preTreeViewItem);
			if (contentPresenter == null)
			{
				return;
			}
			if (contentPresenter != null)
			{
				Grid grid = (Grid)VisualTreeHelper.GetChild(contentPresenter, 0);
				button = grid.FindName("ModifyButton") as Button;
				FrameworkElement frameworkElement = button.Parent as FrameworkElement;
				while (frameworkElement != null && frameworkElement.Name != "XXXXX")
				{
					if (frameworkElement.Name == "stackPanel")
					{
						button2 = frameworkElement.FindName("ApplyButton") as Button;
						button = frameworkElement.FindName("ModifyButton") as Button;
						button3 = frameworkElement.FindName("AddButton") as Button;
						button4 = frameworkElement.FindName("DelButton") as Button;
					}
					frameworkElement = frameworkElement.Parent as FrameworkElement;
				}
				if (frameworkElement == null)
				{
					return;
				}
				textBlock = frameworkElement.FindName("ReadonlyTB") as TextBlock;
				comboBox = frameworkElement.FindName("EditableTB") as ComboBox;
			}
			TblModel tblModel = preTreeViewItem.Header as TblModel;
			if (tblModel != null)
			{
				button.IsEnabled = tblModel.IsEditable;
				button2.Visibility = Visibility.Hidden;
				button.Visibility = Visibility.Visible;
				button4.Visibility = Visibility.Visible;
				button3.Visibility = ((this._selectedTreeLevel == 2) ? Visibility.Collapsed : Visibility.Visible);
			}
			textBlock.Visibility = Visibility.Visible;
			comboBox.Visibility = Visibility.Collapsed;
			comboBox.Text = textBlock.Text;
		}

		// Token: 0x060000B4 RID: 180 RVA: 0x00007FC0 File Offset: 0x000061C0
		internal static void Refresh()
		{
			if (TableSchemaAssociationWindow._tablesTreeView != null)
			{
				TableSchemaAssociationWindow._tablesTreeView.Items.Refresh();
			}
		}

		// Token: 0x060000B5 RID: 181 RVA: 0x00007FD8 File Offset: 0x000061D8
		internal static void RefreshDataSource()
		{
			TableSchemaAssociationWindow._tablesProvider.Refresh();
		}

		// Token: 0x060000B6 RID: 182 RVA: 0x00007FE4 File Offset: 0x000061E4
		private int FindTreeLevel(DependencyObject control)
		{
			int num = -1;
			if (control != null)
			{
				DependencyObject dependencyObject = VisualTreeHelper.GetParent(control);
				while (!(dependencyObject is TreeView) && dependencyObject != null)
				{
					if (dependencyObject is TreeViewItem)
					{
						num++;
					}
					dependencyObject = VisualTreeHelper.GetParent(dependencyObject);
				}
			}
			return num;
		}

		// Token: 0x060000B9 RID: 185 RVA: 0x000080F4 File Offset: 0x000062F4
		[EditorBrowsable(EditorBrowsableState.Never)]
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		[DebuggerNonUserCode]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			switch (connectionId)
			{
			case 3:
				((ComboBox)target).KeyDown += this.EditableTB_KeyDown;
				return;
			case 4:
				((Button)target).Click += this.ApplyButton_Click;
				return;
			case 5:
				((Button)target).Click += this.ModifyButton_Click;
				return;
			default:
				return;
			}
		}

		// Token: 0x0400006A RID: 106
		private Grid grid;

		// Token: 0x0400006B RID: 107
		private Button modifyBtn;

		// Token: 0x0400006C RID: 108
		private Button applyBtn;

		// Token: 0x0400006D RID: 109
		private Button addBtn;

		// Token: 0x0400006E RID: 110
		private Button delBtn;

		// Token: 0x0400006F RID: 111
		private TextBlock readonlyTB;

		// Token: 0x04000070 RID: 112
		private ComboBox editableTB;

		// Token: 0x04000071 RID: 113
		private PackageKey _programKey;

		// Token: 0x04000072 RID: 114
		private IEnumerable<XElement> _tables;

		// Token: 0x04000073 RID: 115
		private static TreeView _tablesTreeView;

		// Token: 0x04000074 RID: 116
		private static ObjectDataProvider _tablesProvider;

		// Token: 0x04000075 RID: 117
		private int _selectedTreeLevel = -1;

		// Token: 0x04000076 RID: 118
		private TreeViewItem currentItem;

		// Token: 0x04000077 RID: 119
		private TreeViewItem preTreeViewItem;
	}
}
