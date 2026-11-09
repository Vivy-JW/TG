using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;

namespace SpecDesigner.FormEditor.DBStructure
{
	// Token: 0x0200005D RID: 93
	public partial class DBStructureUISelector : UserControl, INotifyPropertyChanged
	{
		// Token: 0x0600038F RID: 911 RVA: 0x000137C0 File Offset: 0x000119C0
		public DBStructureUISelector()
		{
			this.InitializeComponent();
		}

		// Token: 0x1700009C RID: 156
		// (get) Token: 0x06000390 RID: 912 RVA: 0x000138D4 File Offset: 0x00011AD4
		public ContainerViewModel Container
		{
			get
			{
				if (this._model == null)
				{
					this._model = new ContainerViewModel();
				}
				return this._model;
			}
		}

		// Token: 0x06000391 RID: 913 RVA: 0x000138F0 File Offset: 0x00011AF0
		private void TopButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.SelectedFields.Source as ObservableCollection<PrepareAddColumn>;
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
			this.DataGridFields.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x06000392 RID: 914 RVA: 0x000139DC File Offset: 0x00011BDC
		private void UpButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.SelectedFields.Source as ObservableCollection<PrepareAddColumn>;
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
			this.DataGridFields.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x06000393 RID: 915 RVA: 0x00013AC0 File Offset: 0x00011CC0
		private void DownButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.SelectedFields.Source as ObservableCollection<PrepareAddColumn>;
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
			this.DataGridFields.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x06000394 RID: 916 RVA: 0x00013BB4 File Offset: 0x00011DB4
		private void BottomButton_Click(object sender, RoutedEventArgs e)
		{
			ObservableCollection<PrepareAddColumn> observableCollection = this.SelectedFields.Source as ObservableCollection<PrepareAddColumn>;
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
			this.DataGridFields.MoveFocus(new TraversalRequest(FocusNavigationDirection.First));
		}

		// Token: 0x1700009D RID: 157
		// (get) Token: 0x06000395 RID: 917 RVA: 0x00013C98 File Offset: 0x00011E98
		public ContainerType[] Containers
		{
			get
			{
				return this._containers;
			}
		}

		// Token: 0x1700009E RID: 158
		// (get) Token: 0x06000396 RID: 918 RVA: 0x00013CA0 File Offset: 0x00011EA0
		// (set) Token: 0x06000397 RID: 919 RVA: 0x00013CB2 File Offset: 0x00011EB2
		public CollectionViewSource SelectedFields
		{
			get
			{
				return (CollectionViewSource)base.GetValue(DBStructureUISelector.SelectedFieldsProperty);
			}
			set
			{
				base.SetValue(DBStructureUISelector.SelectedFieldsProperty, value);
			}
		}

		// Token: 0x1400000B RID: 11
		// (add) Token: 0x06000398 RID: 920 RVA: 0x00013CC0 File Offset: 0x00011EC0
		// (remove) Token: 0x06000399 RID: 921 RVA: 0x00013CF8 File Offset: 0x00011EF8
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600039A RID: 922 RVA: 0x00013D2D File Offset: 0x00011F2D
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x040001D8 RID: 472
		private int _rowCount = 1;

		// Token: 0x040001D9 RID: 473
		private int _columnCount = 1;

		// Token: 0x040001DA RID: 474
		private int _maximumWidth = 30;

		// Token: 0x040001DB RID: 475
		private int _numberOfFields = 2;

		// Token: 0x040001DC RID: 476
		public ContainerViewModel _model;

		// Token: 0x040001DD RID: 477
		private ContainerType[] _containers = new ContainerType[]
		{
			new ContainerType
			{
				Type = "None",
				ImageSource = "images/alignment_LR.png"
			},
			new ContainerType
			{
				Type = "Grid",
				ImageSource = "images/container_grid.png"
			},
			new ContainerType
			{
				Type = "Group",
				ImageSource = "images/widget_groupBox.png"
			},
			new ContainerType
			{
				Type = "ScrollGrid",
				ImageSource = "images/container_scrollgrid.png"
			},
			new ContainerType
			{
				Type = "Table",
				ImageSource = "images/container_table.png"
			},
			new ContainerType
			{
				Type = "Tree",
				ImageSource = "images/widget_tree.png"
			}
		};

		// Token: 0x040001DE RID: 478
		public static readonly DependencyProperty SelectedFieldsProperty = DependencyProperty.Register("SelectedFields", typeof(CollectionViewSource), typeof(DBStructureUISelector), null);
	}
}
