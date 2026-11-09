using System;
using System.CodeDom.Compiler;
using System.Collections;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using System.Xml;
using SpecDesigner.FormDataEditor.Helper;
using SpecDesignerCommon.Connection;

namespace SpecDesigner.FormDataEditor
{
	// Token: 0x02000008 RID: 8
	public partial class MasterDetailView : UserControl
	{
		// Token: 0x14000001 RID: 1
		// (add) Token: 0x06000018 RID: 24 RVA: 0x0000255E File Offset: 0x0000075E
		// (remove) Token: 0x06000019 RID: 25 RVA: 0x0000256C File Offset: 0x0000076C
		public event RoutedEventHandler ItemSelected
		{
			add
			{
				base.AddHandler(MasterDetailView.ItemSelectedEvent, value);
			}
			remove
			{
				base.RemoveHandler(MasterDetailView.ItemSelectedEvent, value);
			}
		}

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x0600001A RID: 26 RVA: 0x0000257A File Offset: 0x0000077A
		// (set) Token: 0x0600001B RID: 27 RVA: 0x0000258C File Offset: 0x0000078C
		public string Source
		{
			get
			{
				return (string)base.GetValue(MasterDetailView.SourceProperty);
			}
			set
			{
				base.SetValue(MasterDetailView.SourceProperty, value);
			}
		}

		// Token: 0x0600001C RID: 28 RVA: 0x0000259A File Offset: 0x0000079A
		private static void SourcePropertyChanged(DependencyObject sender, DependencyPropertyChangedEventArgs e)
		{
		}

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x0600001D RID: 29 RVA: 0x0000259C File Offset: 0x0000079C
		// (set) Token: 0x0600001E RID: 30 RVA: 0x00002600 File Offset: 0x00000800
		public string SelectedID
		{
			get
			{
				if (this._selectedIDs == null || this._selectedIDs.Count == 0)
				{
					return string.Empty;
				}
				SelectionMode selectionType = this.SelectionType;
				if (selectionType == SelectionMode.Multi)
				{
					return string.Join(",", this._selectedIDs.ToArray());
				}
				return this._selectedIDs[this._selectedIDs.Count - 1];
			}
			set
			{
				if (string.IsNullOrWhiteSpace(value))
				{
					return;
				}
				foreach (string text in value.Split(new char[] { ',' }))
				{
					if (this._selectedIDs == null)
					{
						this._selectedIDs = new List<string>();
					}
					if (!string.IsNullOrWhiteSpace(text))
					{
						this._selectedIDs.Add(text.Trim());
					}
				}
			}
		}

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x0600001F RID: 31 RVA: 0x00002668 File Offset: 0x00000868
		// (set) Token: 0x06000020 RID: 32 RVA: 0x00002670 File Offset: 0x00000870
		private SelectionMode SelectionType
		{
			get
			{
				return this._selectType;
			}
			set
			{
				this._selectType = value;
				switch (this._selectType)
				{
				case SelectionMode.Single:
					this.useFlag.Visibility = Visibility.Collapsed;
					this.dataGrid.MouseDoubleClick += this.dataGrid_MouseDoubleClick;
					return;
				case SelectionMode.Multi:
					this.useFlag.Visibility = Visibility.Visible;
					return;
				default:
					return;
				}
			}
		}

		// Token: 0x06000021 RID: 33 RVA: 0x000026CC File Offset: 0x000008CC
		public MasterDetailView()
		{
			this.InitializeComponent();
			this._dataProvider = new XmlDataProvider();
			this._dataProvider.Document = new XmlDocument();
			base.AddHandler(ToggleButton.CheckedEvent, new RoutedEventHandler(this.OnCheckBoxChecked));
			base.AddHandler(ToggleButton.UncheckedEvent, new RoutedEventHandler(this.OnCheckBoxUnchecked));
		}

		// Token: 0x06000022 RID: 34 RVA: 0x0000273C File Offset: 0x0000093C
		public MasterDetailView(string source, int layer)
			: this()
		{
			this._dataProvider.Document.LoadXml(source);
			base.DataContext = this._dataProvider.Document;
			this.Source = source;
			this._layer = layer;
			for (int i = 0; i < layer; i++)
			{
				this.xpath += "/*";
			}
			if (layer == 0)
			{
				this.xpath = "*";
			}
			this.dataGrid.SetBinding(ItemsControl.ItemsSourceProperty, new Binding
			{
				XPath = this.xpath
			});
		}

		// Token: 0x06000023 RID: 35 RVA: 0x000027D3 File Offset: 0x000009D3
		public MasterDetailView(string source, string filterCondition)
			: this(source, 2)
		{
			this.SelectionType = SelectionMode.Single;
			this.query.Text = filterCondition;
			this.Filter();
		}

		// Token: 0x06000024 RID: 36 RVA: 0x00002908 File Offset: 0x00000B08
		public MasterDetailView(string source, SelectionMode mode, string selectedItems)
			: this(source, 2)
		{
			this.SelectionType = mode;
			this.SelectedID = selectedItems;
			base.Loaded += delegate(object s, RoutedEventArgs e)
			{
				if (this._selectedIDs != null && this._selectedIDs.Count > 0)
				{
					IEnumerable itemsSource = this.dataGrid.ItemsSource;
					if (itemsSource == null)
					{
						return;
					}
					foreach (object obj in itemsSource)
					{
						DataGridRow dataGridRow = this.dataGrid.ItemContainerGenerator.ContainerFromItem(obj) as DataGridRow;
						if (dataGridRow != null)
						{
							XmlNode node = dataGridRow.Item as XmlNode;
							if (this._selectedIDs.Any<string>((string id) => id == node.Attributes["id"].Value.Trim()))
							{
								CheckBox visualChild = VisualHelper.GetVisualChild<CheckBox>(dataGridRow);
								if (visualChild != null)
								{
									visualChild.IsChecked = new bool?(true);
								}
							}
						}
					}
				}
			};
		}

		// Token: 0x06000025 RID: 37 RVA: 0x00002944 File Offset: 0x00000B44
		public MasterDetailView(string source)
			: this(source, SelectionMode.Single, string.Empty)
		{
		}

		// Token: 0x06000026 RID: 38 RVA: 0x00002954 File Offset: 0x00000B54
		private void dataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			IInputElement directlyOver = e.MouseDevice.DirectlyOver;
			if (directlyOver != null && directlyOver is FrameworkElement && ((FrameworkElement)directlyOver).Parent is DataGridCell)
			{
				DataGrid dataGrid = sender as DataGrid;
				if (dataGrid != null && dataGrid.SelectedItems != null && dataGrid.SelectedItems.Count == 1)
				{
					XmlNode xmlNode = dataGrid.SelectedItem as XmlNode;
					if (xmlNode == null)
					{
						return;
					}
					this.SelectedID = xmlNode.Attributes["id"].Value;
					RoutedEventArgs e2 = new RoutedEventArgs(MasterDetailView.ItemSelectedEvent);
					base.RaiseEvent(e2);
				}
			}
		}

		// Token: 0x06000027 RID: 39 RVA: 0x000029E7 File Offset: 0x00000BE7
		private void SearchButton_Click(object sender, RoutedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x06000028 RID: 40 RVA: 0x00002A58 File Offset: 0x00000C58
		private void Filter()
		{
			if (string.IsNullOrEmpty(this.query.Text))
			{
				return;
			}
			this._customerView = CollectionViewSource.GetDefaultView(this.dataGrid.ItemsSource);
			this._customerView.Filter = (object item) => ((XmlNode)item).Attributes["id"].Value.Contains(this.query.Text) || ((XmlNode)item).Attributes["desc"].Value.Contains(this.query.Text);
			this.dataGrid.SelectedIndex = 0;
		}

		// Token: 0x06000029 RID: 41 RVA: 0x00002AB1 File Offset: 0x00000CB1
		private void ResetList()
		{
			if (this._customerView != null)
			{
				this._customerView.Filter = null;
			}
			this.dataGrid.SelectedIndex = 0;
		}

		// Token: 0x0600002A RID: 42 RVA: 0x00002AD3 File Offset: 0x00000CD3
		private void query_TextChanged(object sender, TextChangedEventArgs e)
		{
			if (string.IsNullOrEmpty(this.query.Text))
			{
				this.ResetList();
				return;
			}
			this.Filter();
		}

		// Token: 0x0600002B RID: 43 RVA: 0x00002AF4 File Offset: 0x00000CF4
		private void OnCheckBoxChecked(object sender, RoutedEventArgs e)
		{
			string text = (e.OriginalSource as CheckBox).Tag as string;
			if (!this._selectedIDs.Contains(text))
			{
				this._selectedIDs.Add(text);
			}
		}

		// Token: 0x0600002C RID: 44 RVA: 0x00002B34 File Offset: 0x00000D34
		private void OnCheckBoxUnchecked(object sender, RoutedEventArgs e)
		{
			string text = (e.OriginalSource as CheckBox).Tag as string;
			if (this._selectedIDs.Contains(text))
			{
				this._selectedIDs.Remove(text);
			}
		}

		// Token: 0x0600002D RID: 45 RVA: 0x00002B74 File Offset: 0x00000D74
		private void queryButton_Click(object sender, RoutedEventArgs e)
		{
			XmlNode xmlNode = this.dataGrid.SelectedItem as XmlNode;
			if (xmlNode == null)
			{
				return;
			}
			string value = xmlNode.Attributes["id"].Value;
			string text = string.Empty;
			if (xmlNode.Name == "zoom")
			{
				text = "adzi210";
			}
			else
			{
				if (!(xmlNode.Name == "check"))
				{
					return;
				}
				text = "adzi220";
			}
			ConnectionManager.RunProgram(string.Format("{0} {1}", text, value));
		}

		// Token: 0x0600002E RID: 46 RVA: 0x00002BFC File Offset: 0x00000DFC
		private void callButton_Click(object sender, RoutedEventArgs e)
		{
			XmlNode xmlNode = this.dataGrid.SelectedItem as XmlNode;
			if (xmlNode == null)
			{
				return;
			}
			string text = string.Empty;
			if (xmlNode.Name == "zoom")
			{
				text = "adzi210";
			}
			else
			{
				if (!(xmlNode.Name == "check"))
				{
					return;
				}
				text = "adzi220";
			}
			ConnectionManager.RunProgram(text);
		}

		// Token: 0x0600002F RID: 47 RVA: 0x00002C60 File Offset: 0x00000E60
		private void dataGrid_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			this.statusItem.Content = string.Format("List Count: {0}, Selected: {1}", this.dataGrid.Items.Count, this.dataGrid.SelectedIndex + 1);
		}

		// Token: 0x06000035 RID: 53 RVA: 0x00002DE4 File Offset: 0x00000FE4
		// Note: this type is marked as 'beforefieldinit'.
		static MasterDetailView()
		{
			MasterDetailView.ItemSelectedEvent = EventManager.RegisterRoutedEvent("ItemSelected", RoutingStrategy.Bubble, typeof(RoutedEventHandler), typeof(MasterDetailView));
			MasterDetailView.SourceProperty = DependencyProperty.Register("Source", typeof(string), typeof(MasterDetailView), new PropertyMetadata(new PropertyChangedCallback(MasterDetailView.SourcePropertyChanged)));
		}

		// Token: 0x04000009 RID: 9
		private XmlDataProvider _dataProvider;

		// Token: 0x0400000A RID: 10
		public static DependencyProperty SourceProperty;

		// Token: 0x0400000B RID: 11
		private int _layer;

		// Token: 0x0400000C RID: 12
		private string xpath;

		// Token: 0x0400000D RID: 13
		private List<string> _selectedIDs = new List<string>();

		// Token: 0x0400000E RID: 14
		private SelectionMode _selectType;

		// Token: 0x0400000F RID: 15
		private ICollectionView _customerView;
	}
}
