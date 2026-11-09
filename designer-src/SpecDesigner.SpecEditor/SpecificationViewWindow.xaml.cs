using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Media;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesigner.SpecEditor.Helpers;
using SpecDesigner.SpecEditor.ViewModel;
using SpecDesigner.SpecEditor.Views;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor
{
	// Token: 0x02000016 RID: 22
	public partial class SpecificationViewWindow : Window
	{
		// Token: 0x17000010 RID: 16
		// (get) Token: 0x0600008F RID: 143 RVA: 0x00006D9D File Offset: 0x00004F9D
		// (set) Token: 0x06000090 RID: 144 RVA: 0x00006DA5 File Offset: 0x00004FA5
		private PackageKey ProgramKey { get; set; }

		// Token: 0x06000091 RID: 145 RVA: 0x00006DAE File Offset: 0x00004FAE
		private SpecificationViewWindow()
		{
			this.InitializeComponent();
			base.CommandBindings.Add(new CommandBinding(SpecPropertyCommands.ShowZoomsWindowCommand, new ExecutedRoutedEventHandler(SpecPropertyCommands.ExecutedShowZoomsWindow), new CanExecuteRoutedEventHandler(SpecPropertyCommands.CanShowZoomsWindow)));
		}

		// Token: 0x06000092 RID: 146 RVA: 0x00006DEA File Offset: 0x00004FEA
		public SpecificationViewWindow(PackageKey programKey)
			: this(programKey, false)
		{
		}

		// Token: 0x06000093 RID: 147 RVA: 0x00006DF4 File Offset: 0x00004FF4
		public SpecificationViewWindow(PackageKey programKey, bool isUncitedOnly)
			: this()
		{
			this.ProgramKey = programKey;
			this.IsUncitedOnly = isUncitedOnly;
			base.Title = (this.IsUncitedOnly ? string.Format("{0} [{1}]", base.FindResource("menu_UncitedSpecificationView") as string, this.ProgramKey.Program) : string.Format("{0} [{1}]", base.FindResource("menu_SpecificationView") as string, this.ProgramKey.Program));
			this.SetData();
			if (this.IsUncitedOnly)
			{
				this.excludeTabItem.Visibility = Visibility.Collapsed;
			}
		}

		// Token: 0x17000011 RID: 17
		// (get) Token: 0x06000094 RID: 148 RVA: 0x00006E89 File Offset: 0x00005089
		// (set) Token: 0x06000095 RID: 149 RVA: 0x00006E91 File Offset: 0x00005091
		[DefaultValue(false)]
		public bool IsUncitedOnly { get; set; }

		// Token: 0x06000096 RID: 150 RVA: 0x00006E9A File Offset: 0x0000509A
		public new void Show()
		{
			base.Owner = Application.Current.MainWindow;
			base.ShowDialog();
		}

		// Token: 0x06000097 RID: 151 RVA: 0x00006EB3 File Offset: 0x000050B3
		protected override void OnClosing(CancelEventArgs e)
		{
			base.OnClosing(e);
			this.CommitEdit();
		}

		// Token: 0x06000098 RID: 152 RVA: 0x00006EC4 File Offset: 0x000050C4
		private void SetData()
		{
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo;
			this.specFieldsTabItem.DataContext = new SpecNodesSourceViewModel<SpecFieldNode>(specificationInfo.FieldsForView, this.IsUncitedOnly);
			this.multiLangTabItem.DataContext = new SpecNodesSourceViewModel<SpecMultiLangNode>(specificationInfo.MultiLangsForView, this.IsUncitedOnly);
			this.referenceTabItem.DataContext = new SpecNodesSourceViewModel<SpecReferenceNode>(specificationInfo.ReferenceNodesForView, this.IsUncitedOnly);
			this.progRelTabItem.DataContext = new SpecNodesSourceViewModel<SpecProgRelNode>(specificationInfo.ProgRelNodesForView, this.IsUncitedOnly);
			this.actionsTabItem.DataContext = new SpecNodesSourceViewModel<SpecActionNode>(specificationInfo.ActionsForView, this.IsUncitedOnly);
			this.treeTabItem.DataContext = new SpecNodesSourceViewModel<SpecTreeNode>(specificationInfo.TreesForView, this.IsUncitedOnly);
			this.excludeGrid.Children.Add(new SpecExcludeView(this.ProgramKey));
			if (specificationInfo.Env.Equals("c", StringComparison.CurrentCultureIgnoreCase))
			{
				this.specCustCol.Visibility = (this.langCustCol.Visibility = (this.refCustCol.Visibility = (this.progCustCol.Visibility = (this.actCustCol.Visibility = Visibility.Visible))));
				return;
			}
			this.specCustCol.Visibility = (this.langCustCol.Visibility = (this.refCustCol.Visibility = (this.progCustCol.Visibility = (this.actCustCol.Visibility = Visibility.Collapsed))));
		}

		// Token: 0x06000099 RID: 153 RVA: 0x00007054 File Offset: 0x00005254
		private void CommitEdit()
		{
			foreach (DataGrid dataGrid in SpecificationViewWindow.FindVisualChildren<DataGrid>(this))
			{
				IEditableCollectionView items = dataGrid.Items;
				if (items.IsEditingItem)
				{
					dataGrid.CommitEdit();
				}
			}
		}

		// Token: 0x0600009A RID: 154 RVA: 0x000072E4 File Offset: 0x000054E4
		public static IEnumerable<T> FindVisualChildren<T>(DependencyObject depObj) where T : DependencyObject
		{
			if (depObj != null)
			{
				for (int i = 0; i < VisualTreeHelper.GetChildrenCount(depObj); i++)
				{
					DependencyObject child = VisualTreeHelper.GetChild(depObj, i);
					if (child != null && child is T)
					{
						yield return (T)((object)child);
					}
					foreach (T childOfChild in SpecificationViewWindow.FindVisualChildren<T>(child))
					{
						yield return childOfChild;
					}
				}
			}
			yield break;
		}

		// Token: 0x0600009B RID: 155 RVA: 0x00007318 File Offset: 0x00005518
		private void ExportActionButton_Click(object sender, RoutedEventArgs e)
		{
			string text = string.Format("C:/TT/{0}_Specification_action.csv", this.ProgramKey);
			bool flag = ExportTools.ExportXElementToCSV((from ele in XElement.Parse(SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo.ToString()).Elements()
				where ele.Name == "act"
				select ele).ToList<XElement>(), text);
			if (flag)
			{
				MessageBox.Show(string.Format("{0} 輸出完成", text));
			}
		}

		// Token: 0x0600009C RID: 156 RVA: 0x000073B4 File Offset: 0x000055B4
		private void ExportFieldButton_Click(object sender, RoutedEventArgs e)
		{
			string text = string.Format("C:/TT/{0}_Specfication_field.csv", this.ProgramKey);
			bool flag = ExportTools.ExportXElementToCSV((from ele in XElement.Parse(SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo.ToString()).Elements()
				where ele.Name == "field"
				select ele).ToList<XElement>(), text);
			if (flag)
			{
				DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_ExportSucess") as string, text));
			}
		}

		// Token: 0x0600009D RID: 157 RVA: 0x00007448 File Offset: 0x00005648
		private void DataGrid_BeginningEdit(object sender, DataGridBeginningEditEventArgs e)
		{
			AbstractSpecNode abstractSpecNode = e.Row.Item as AbstractSpecNode;
			e.Cancel = true;
			if (abstractSpecNode != null)
			{
				e.Cancel = abstractSpecNode.IsCited;
			}
		}

		// Token: 0x0600009E RID: 158 RVA: 0x0000747C File Offset: 0x0000567C
		private void TreeNode_BeginningEdit(object sender, DataGridBeginningEditEventArgs e)
		{
			SpecTreeNode specTreeNode = e.Row.Item as SpecTreeNode;
			if (specTreeNode != null)
			{
				string[] array = specTreeNode.Att.Split(new char[] { ',' });
				e.Cancel = !array.Contains(e.Column.Header);
			}
		}
	}
}
