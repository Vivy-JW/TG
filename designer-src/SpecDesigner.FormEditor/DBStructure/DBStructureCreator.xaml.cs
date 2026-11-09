using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.FormEditor.DBStructure
{
	// Token: 0x02000028 RID: 40
	public partial class DBStructureCreator : Window
	{
		// Token: 0x0600015F RID: 351 RVA: 0x0000792F File Offset: 0x00005B2F
		public DBStructureCreator(PackageKey programKey)
		{
			this.InitializeComponent();
			base.Loaded += this.DBStructureCreator_Loaded;
			this.ProgramKey = programKey;
		}

		// Token: 0x17000049 RID: 73
		// (get) Token: 0x06000160 RID: 352 RVA: 0x00007961 File Offset: 0x00005B61
		// (set) Token: 0x06000161 RID: 353 RVA: 0x00007969 File Offset: 0x00005B69
		public FrameworkElement LayoutEditor { get; set; }

		// Token: 0x1700004A RID: 74
		// (get) Token: 0x06000162 RID: 354 RVA: 0x00007972 File Offset: 0x00005B72
		// (set) Token: 0x06000163 RID: 355 RVA: 0x0000797A File Offset: 0x00005B7A
		public PackageKey ProgramKey { get; set; }

		// Token: 0x06000164 RID: 356 RVA: 0x00007984 File Offset: 0x00005B84
		private void AddAllBtn_Click(object sender, RoutedEventArgs e)
		{
			foreach (object obj in this.ColumnDataGrid.ItemsSource)
			{
				XElement xelement = (XElement)obj;
				this.addSelection(xelement);
			}
		}

		// Token: 0x06000165 RID: 357 RVA: 0x000079E4 File Offset: 0x00005BE4
		private void AddBtn_Click(object sender, RoutedEventArgs e)
		{
			foreach (object obj in this.ColumnDataGrid.SelectedItems)
			{
				XElement xelement = (XElement)obj;
				if (xelement != null && xelement.Name.LocalName == "column")
				{
					this.addSelection(xelement);
				}
			}
			this.ColumnDataGrid.SelectedItems.Clear();
		}

		// Token: 0x06000166 RID: 358 RVA: 0x00007A6C File Offset: 0x00005C6C
		private void addSelection(XElement item)
		{
			PrepareAddColumn prepareAddColumn;
			this._selectedColumn.Add(prepareAddColumn = new PrepareAddColumn
			{
				Table = (this.TableDataGrid.SelectedItem as XElement).Attribute("name").Value,
				Column = item.Attribute("name").Value,
				Description = item.Attribute("text").Value,
				Label = item.Attribute("text").Value
			});
			XElement colField = TableColumnHelper.GetColField(prepareAddColumn.Table, prepareAddColumn.Column);
			if (colField != null)
			{
				prepareAddColumn.Widget = colField.Attribute("widget").Value;
				prepareAddColumn.Width = colField.Attribute("widget_width").Value;
			}
		}

		// Token: 0x06000167 RID: 359 RVA: 0x00007B58 File Offset: 0x00005D58
		private void BottomButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.selectedSource.Source as ObservableCollection<PrepareAddColumn>;
			List<PrepareAddColumn> list = new List<PrepareAddColumn>();
			foreach (PrepareAddColumn prepareAddColumn in observableCollection)
			{
				if (prepareAddColumn.IsSelected)
				{
					list.Add(prepareAddColumn);
				}
				prepareAddColumn.IsSelected = false;
			}
			foreach (object obj in list)
			{
				PrepareAddColumn prepareAddColumn2 = obj as PrepareAddColumn;
				observableCollection.Remove(prepareAddColumn2);
				observableCollection.Add(prepareAddColumn2);
				prepareAddColumn2.IsSelected = true;
			}
			list.Clear();
			this.SelectionDataGrid.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x06000168 RID: 360 RVA: 0x00007C3C File Offset: 0x00005E3C
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.Close();
		}

		// Token: 0x06000169 RID: 361 RVA: 0x00007C44 File Offset: 0x00005E44
		private void ColumnDataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			if (sender != null)
			{
				DataGrid dataGrid = sender as DataGrid;
				if (dataGrid != null && dataGrid.SelectedItems != null && dataGrid.SelectedItems.Count == 1)
				{
					this.addSelection(dataGrid.SelectedItem as XElement);
				}
			}
		}

		// Token: 0x0600016A RID: 362 RVA: 0x00007C88 File Offset: 0x00005E88
		private void DBStructureCreator_Loaded(object sender, RoutedEventArgs e)
		{
			if (this._tables == null)
			{
				this._tables = TableColumnHelper.GetOrderedTables(SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo);
				this.viewSource = base.FindResource("tablesSource") as CollectionViewSource;
				this.viewSource.Source = this._tables;
				this.viewSource.Filter += this.viewSource_Filter;
			}
			if (this.selectedSource == null)
			{
				this.selectedSource = base.FindResource("SelectedColumnSource") as CollectionViewSource;
				this.selectedSource.Source = this._selectedColumn;
			}
		}

		// Token: 0x0600016B RID: 363 RVA: 0x00007D2C File Offset: 0x00005F2C
		private void DownButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.selectedSource.Source as ObservableCollection<PrepareAddColumn>;
			List<PrepareAddColumn> list = new List<PrepareAddColumn>();
			foreach (PrepareAddColumn prepareAddColumn in observableCollection)
			{
				if (prepareAddColumn.IsSelected)
				{
					list.Add(prepareAddColumn);
				}
				prepareAddColumn.IsSelected = false;
			}
			list.Reverse();
			foreach (PrepareAddColumn prepareAddColumn2 in list)
			{
				int num = observableCollection.IndexOf(prepareAddColumn2);
				if (num < observableCollection.Count - 1)
				{
					observableCollection.Move(num, num + 1);
				}
				prepareAddColumn2.IsSelected = true;
			}
			list.Clear();
			this.SelectionDataGrid.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x0600016C RID: 364 RVA: 0x00007E20 File Offset: 0x00006020
		private void FinishButton_Click(object sender, RoutedEventArgs e)
		{
			if ((this.uiSelector.SelectedFields.Source as IEnumerable<PrepareAddColumn>).Count<PrepareAddColumn>() == 0)
			{
				return;
			}
			base.Close();
			UICreator.Create(this.LayoutEditor, this.uiSelector.Container, this.uiSelector.SelectedFields.Source as IEnumerable<PrepareAddColumn>, this.ProgramKey);
		}

		// Token: 0x0600016D RID: 365 RVA: 0x00007E81 File Offset: 0x00006081
		private void RemoveAllBtn_Click(object sender, RoutedEventArgs e)
		{
			this._selectedColumn.Clear();
		}

		// Token: 0x0600016E RID: 366 RVA: 0x00007E90 File Offset: 0x00006090
		private void RemoveBtn_Click(object sender, RoutedEventArgs e)
		{
			List<PrepareAddColumn> list = new List<PrepareAddColumn>();
			foreach (object obj in this.SelectionDataGrid.SelectedItems)
			{
				PrepareAddColumn prepareAddColumn = (PrepareAddColumn)obj;
				list.Add(prepareAddColumn);
			}
			foreach (PrepareAddColumn prepareAddColumn2 in list)
			{
				this._selectedColumn.Remove(prepareAddColumn2);
			}
		}

		// Token: 0x0600016F RID: 367 RVA: 0x00007F3C File Offset: 0x0000613C
		private void SelectionDataGrid_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.selectedSource.Source as ObservableCollection<PrepareAddColumn>;
			this.UpButton.IsEnabled = true;
			this.TopButton.IsEnabled = true;
			this.DownButton.IsEnabled = true;
			this.BottomButton.IsEnabled = true;
			foreach (PrepareAddColumn prepareAddColumn in observableCollection)
			{
				if (prepareAddColumn.IsSelected)
				{
					int num = observableCollection.IndexOf(prepareAddColumn);
					if (num == 0)
					{
						this.TopButton.IsEnabled = false;
						this.UpButton.IsEnabled = false;
					}
					if (num == observableCollection.Count - 1)
					{
						this.DownButton.IsEnabled = false;
						this.BottomButton.IsEnabled = false;
					}
				}
			}
		}

		// Token: 0x06000170 RID: 368 RVA: 0x00008010 File Offset: 0x00006210
		private void TableDataGrid_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			IEnumerable<XElement> enumerable = null;
			if (e.AddedItems.Count == 1)
			{
				enumerable = TableColumnHelper.FindTableColumns((e.AddedItems[0] as XElement).Attribute("name").Value, SettingManager.Get().GetTzpManger(this.ProgramKey).SpecificationInfo.Env);
			}
			this.ColumnDataGrid.ItemsSource = enumerable;
		}

		// Token: 0x06000171 RID: 369 RVA: 0x0000807E File Offset: 0x0000627E
		private void tableFilterTB_TextChanged(object sender, TextChangedEventArgs e)
		{
			if (this.viewSource != null)
			{
				this.viewSource.View.Refresh();
			}
		}

		// Token: 0x06000172 RID: 370 RVA: 0x00008098 File Offset: 0x00006298
		private void TopButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.selectedSource.Source as ObservableCollection<PrepareAddColumn>;
			List<PrepareAddColumn> list = new List<PrepareAddColumn>();
			foreach (PrepareAddColumn prepareAddColumn in observableCollection)
			{
				if (prepareAddColumn.IsSelected)
				{
					list.Add(prepareAddColumn);
				}
				prepareAddColumn.IsSelected = false;
			}
			int num = 0;
			foreach (object obj in list)
			{
				PrepareAddColumn prepareAddColumn2 = obj as PrepareAddColumn;
				observableCollection.Remove(prepareAddColumn2);
				observableCollection.Insert(num, prepareAddColumn2);
				prepareAddColumn2.IsSelected = true;
				num++;
			}
			list.Clear();
			this.SelectionDataGrid.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x06000173 RID: 371 RVA: 0x00008184 File Offset: 0x00006384
		private void UpButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.selectedSource.Source as ObservableCollection<PrepareAddColumn>;
			List<PrepareAddColumn> list = new List<PrepareAddColumn>();
			foreach (PrepareAddColumn prepareAddColumn in observableCollection)
			{
				if (prepareAddColumn.IsSelected)
				{
					list.Add(prepareAddColumn);
				}
				prepareAddColumn.IsSelected = false;
			}
			foreach (PrepareAddColumn prepareAddColumn2 in list)
			{
				int num = observableCollection.IndexOf(prepareAddColumn2);
				if (num > 0)
				{
					observableCollection.Move(num, num - 1);
				}
				prepareAddColumn2.IsSelected = true;
			}
			list.Clear();
			this.SelectionDataGrid.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x06000174 RID: 372 RVA: 0x00008268 File Offset: 0x00006468
		private void viewSource_Filter(object sender, FilterEventArgs e)
		{
			XElement xelement = e.Item as XElement;
			if (this.tableFilterTB.Text != "")
			{
				e.Accepted = (xelement.Attribute("name").Value + xelement.Attribute("desc").Value + xelement.Attribute("module").Value).IndexOf(this.tableFilterTB.Text, StringComparison.CurrentCultureIgnoreCase) != -1;
			}
		}

		// Token: 0x040000C0 RID: 192
		private ObservableCollection<PrepareAddColumn> _selectedColumn = new ObservableCollection<PrepareAddColumn>();

		// Token: 0x040000C1 RID: 193
		private IEnumerable<XElement> _tables;

		// Token: 0x040000C2 RID: 194
		private CollectionViewSource selectedSource;

		// Token: 0x040000C3 RID: 195
		private CollectionViewSource viewSource;
	}
}
