using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.ActionDefaults
{
	// Token: 0x0200000E RID: 14
	public partial class ActionDefaults : UserControl, IDisposable
	{
		// Token: 0x06000040 RID: 64 RVA: 0x00003590 File Offset: 0x00001790
		public ActionDefaults()
		{
			this.InitializeComponent();
			this.BindingCommands();
			base.DataContextChanged += this.ActionDefaults_DataContextChanged;
			base.Loaded += this.ActionDefaults_Loaded;
		}

		// Token: 0x06000041 RID: 65 RVA: 0x000035E0 File Offset: 0x000017E0
		private void ActionDefaults_Loaded(object sender, RoutedEventArgs e)
		{
			if (this._viewSource == null)
			{
				this._viewSource = CollectionViewSource.GetDefaultView(this.actionList.ItemsSource);
			}
			this._viewSource.SortDescriptions.Add(new SortDescription("Name", ListSortDirection.Ascending));
			this.Filter();
		}

		// Token: 0x06000042 RID: 66 RVA: 0x0000362C File Offset: 0x0000182C
		private void ActionDefaults_DataContextChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			base.DataContextChanged -= this.ActionDefaults_DataContextChanged;
			this.UpdateGroups();
		}

		// Token: 0x06000043 RID: 67 RVA: 0x00003648 File Offset: 0x00001848
		private void UpdateGroups()
		{
			SpecificationInfo specificationInfo = base.DataContext as SpecificationInfo;
			if (specificationInfo == null)
			{
				return;
			}
			this._groupList.Clear();
			this._groupList.Add(new ActionDefaults.ActType("", ""));
			this._groupList.Add(new ActionDefaults.ActType("all", Application.Current.FindResource("ads_all") as string));
			this._groupList.Add(new ActionDefaults.ActType("mi", Application.Current.FindResource("ads_mi") as string));
			foreach (FormSpecModel formSpecModel in specificationInfo.FormSpeDictionary.Values)
			{
				if (formSpecModel.Name.StartsWith("s_detail"))
				{
					if (Regex.IsMatch(formSpecModel.Name, "^s_detail\\d+_info$"))
					{
						return;
					}
					this._groupList.Add(new ActionDefaults.ActType(string.Format("d[ib]{0}", formSpecModel.Name.Replace("s_detail", "")), formSpecModel.Name));
				}
			}
			this.groupSelector.ItemsSource = this._groupList;
			this.groupSelector.SelectedIndex = 0;
		}

		// Token: 0x06000044 RID: 68 RVA: 0x00003798 File Offset: 0x00001998
		private void BindingCommands()
		{
			base.CommandBindings.Add(new CommandBinding(ActionDefaultCommands.AddActionCommand, new ExecutedRoutedEventHandler(this.ExecutedAddAction)));
			base.CommandBindings.Add(new CommandBinding(ActionDefaultCommands.DeleteActionCommand, new ExecutedRoutedEventHandler(ActionDefaultCommands.ExecutedDeleteAction), new CanExecuteRoutedEventHandler(ActionDefaultCommands.CanExecuteDeleteAction)));
		}

		// Token: 0x06000045 RID: 69 RVA: 0x000037F8 File Offset: 0x000019F8
		private void ExecutedAddAction(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger((PackageKey)e.Parameter).SpecificationInfo;
			SpecActionNode specActionNode = SpecActionNode.Create(specificationInfo);
			specificationInfo.Add(specActionNode);
			this.groupSelector.SelectedIndex = 0;
			this.searchBox.Text = string.Empty;
		}

		// Token: 0x06000046 RID: 70 RVA: 0x00003851 File Offset: 0x00001A51
		private void searchBox_TextChanged(object sender, TextChangedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x06000047 RID: 71 RVA: 0x00003859 File Offset: 0x00001A59
		private void groupSelector_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x06000048 RID: 72 RVA: 0x000038F0 File Offset: 0x00001AF0
		private void Filter()
		{
			string name = this.searchBox.Text.Trim();
			string type = ((this.groupSelector.SelectedValue is ActionDefaults.ActType) ? (this.groupSelector.SelectedValue as ActionDefaults.ActType).Type : null);
			if (this._viewSource == null || type == null)
			{
				return;
			}
			this._viewSource.Filter = delegate(object item)
			{
				SpecActionNode specActionNode = item as SpecActionNode;
				return specActionNode != null && !specActionNode.IsToolBarAction && (specActionNode.Status & SpecStatus.DELETE) != SpecStatus.DELETE && (string.IsNullOrEmpty(type) || specActionNode.IsContainsType(type)) && (string.IsNullOrWhiteSpace(name) || specActionNode.Name.Contains(name) || specActionNode.LocalString.Contains(name));
			};
		}

		// Token: 0x06000049 RID: 73 RVA: 0x00003971 File Offset: 0x00001B71
		public void Dispose()
		{
			base.Loaded -= this.ActionDefaults_Loaded;
			base.DataContextChanged -= this.ActionDefaults_DataContextChanged;
			base.CommandBindings.Clear();
			this._viewSource = null;
		}

		// Token: 0x0400002F RID: 47
		private List<ActionDefaults.ActType> _groupList = new List<ActionDefaults.ActType>();

		// Token: 0x04000030 RID: 48
		private ICollectionView _viewSource;

		// Token: 0x0200000F RID: 15
		private class ActType
		{
			// Token: 0x17000003 RID: 3
			// (get) Token: 0x0600004C RID: 76 RVA: 0x00003A6E File Offset: 0x00001C6E
			// (set) Token: 0x0600004D RID: 77 RVA: 0x00003A76 File Offset: 0x00001C76
			public string Type { get; private set; }

			// Token: 0x17000004 RID: 4
			// (get) Token: 0x0600004E RID: 78 RVA: 0x00003A7F File Offset: 0x00001C7F
			// (set) Token: 0x0600004F RID: 79 RVA: 0x00003A87 File Offset: 0x00001C87
			public string Text { get; private set; }

			// Token: 0x06000050 RID: 80 RVA: 0x00003A90 File Offset: 0x00001C90
			public ActType(string type, string text)
			{
				this.Type = type;
				this.Text = text;
			}
		}
	}
}
