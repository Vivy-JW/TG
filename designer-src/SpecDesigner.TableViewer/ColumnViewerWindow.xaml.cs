using System;
using System.CodeDom.Compiler;
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
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.TableViewer
{
	// Token: 0x02000006 RID: 6
	public partial class ColumnViewerWindow : Window, IDisposable
	{
		// Token: 0x17000002 RID: 2
		// (get) Token: 0x06000024 RID: 36 RVA: 0x00002DBE File Offset: 0x00000FBE
		public static ColumnViewerWindow This
		{
			get
			{
				if (ColumnViewerWindow._this == null)
				{
					ColumnViewerWindow._this = new ColumnViewerWindow();
				}
				return ColumnViewerWindow._this;
			}
		}

		// Token: 0x06000025 RID: 37 RVA: 0x00002DD8 File Offset: 0x00000FD8
		private ColumnViewerWindow()
		{
			this.InitializeComponent();
			base.Title = Application.Current.FindResource("columnView_WindowTitle") as string;
			this.LoadTableSchema();
			this.tableView.DataContext = this._tables;
			this._filterView = CollectionViewSource.GetDefaultView(this.tableView.Items);
			this.moduleCombo.DataContext = this._modules;
			this.tableView.SelectionChanged += this.tableView_SelectionChanged;
			this.moduleCombo.PreviewKeyDown += this.Searc_PreviewKeyDown;
			this.moduleCombo.SelectionChanged += this.moduleCombo_SelectionChanged;
			this.searchBox.TextChanged += this.searchBox_TextChanged;
			Application.Current.Exit += this.Current_Exit;
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Subscribe(new Action<string>(this.Reload));
			this.BindCommands();
		}

		// Token: 0x06000026 RID: 38 RVA: 0x00002EF4 File Offset: 0x000010F4
		private void BindCommands()
		{
			base.CommandBindings.Add(new CommandBinding(TableColumnCommands.CopyAllColumnsNameCommand, new ExecutedRoutedEventHandler(TableColumnCommands.ExecutedCopyAllColumnsName), new CanExecuteRoutedEventHandler(TableColumnCommands.CanExecuteCopyAllColumnsName)));
			base.CommandBindings.Add(new CommandBinding(TableColumnCommands.CopyAllColumnsToRecordCommand, new ExecutedRoutedEventHandler(TableColumnCommands.ExecutedCopyAllColumnsToRecord), new CanExecuteRoutedEventHandler(TableColumnCommands.CanExecuteCopyAllColumnsToRecord)));
		}

		// Token: 0x06000027 RID: 39 RVA: 0x00002F5D File Offset: 0x0000115D
		private void Current_Exit(object sender, ExitEventArgs e)
		{
			Application.Current.Exit -= this.Current_Exit;
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Unsubscribe(new Action<string>(this.Reload));
			base.Close();
		}

		// Token: 0x06000028 RID: 40 RVA: 0x00002F98 File Offset: 0x00001198
		private void Reload(object nullObj)
		{
			string text = this.searchBox.Text;
			this.searchBox.Text = string.Empty;
			this.LoadTableSchema();
			if (!string.IsNullOrEmpty(text))
			{
				this.searchBox.Text = text;
				this.Filter();
			}
		}

		// Token: 0x06000029 RID: 41 RVA: 0x00003008 File Offset: 0x00001208
		private void LoadTableSchema()
		{
			if (this._tables.Count > 0)
			{
				this._tables.Clear();
			}
			foreach (XElement xelement in TableColumnHelper.GetTables())
			{
				this._tables.Add(xelement);
			}
			IEnumerable<IGrouping<string, XElement>> enumerable = from t in this._tables
				group t by t.Attribute("module").Value into g
				where !string.IsNullOrWhiteSpace(g.Key)
				select g;
			foreach (IGrouping<string, XElement> grouping in enumerable)
			{
				this._modules.Add(grouping.Key);
			}
		}

		// Token: 0x0600002A RID: 42 RVA: 0x0000310C File Offset: 0x0000130C
		public new void Show()
		{
			this.LoadTableSchema();
			PackageKey packageKey = (PackageKey)Application.Current.MainWindow.Tag;
			if (packageKey != null && SettingManager.Get().GetTzpManger(packageKey).Type == TzpType.Form && this.searchBox.Text.Length == 0)
			{
				XElement source = SettingManager.Get().GetTzpManger(packageKey).SpecificationInfo.AssociateTable.Source;
				string[] array = (from attr in source.Elements("tbl").Attributes("name")
					select attr.Value).ToArray<string>();
				string text = string.Join("|", array);
				this.searchBox.Text = text;
			}
			base.Show();
			if (base.Visibility != Visibility.Visible)
			{
				base.Visibility = Visibility.Visible;
			}
			base.Focus();
		}

		// Token: 0x0600002B RID: 43 RVA: 0x000031FE File Offset: 0x000013FE
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
			this._tables.Clear();
			this._modules.Clear();
		}

		// Token: 0x0600002C RID: 44 RVA: 0x00003224 File Offset: 0x00001424
		private void tableView_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			XElement xelement = this.tableView.SelectedItem as XElement;
			if (xelement == null)
			{
				this.SetColumnContent(null);
				this.tableLabel.Text = string.Empty;
			}
			else
			{
				this.SetColumnContent(xelement.Attribute("name").Value);
				this.tableLabel.Text = string.Format(base.FindResource("tableView_ColumnTitle") as string, xelement.Attribute("name").Value, xelement.Attribute("desc").Value);
			}
			this.statusItem.Content = string.Format("Table Count: {0}, Selected: {1}", this.tableView.Items.Count, this.tableView.SelectedIndex + 1);
		}

		// Token: 0x0600002D RID: 45 RVA: 0x00003384 File Offset: 0x00001584
		private void SetColumnContent(string tableName)
		{
			this.columnView.DataContext = null;
			this.langView.DataContext = null;
			this.refView.DataContext = null;
			this.statusView.DataContext = null;
			this.treeView.DataContext = null;
			if (!string.IsNullOrWhiteSpace(tableName))
			{
				XElement xelement = new XElement("col_attr");
				IEnumerable<XElement> colFields = TableColumnHelper.GetColFields(tableName);
				if (colFields == null)
				{
					DesignerMessageBox.Show(Application.Current.FindResource("Message_MissBasicData") as string, Application.Current.FindResource("Message_Message") as string, MessageBoxButton.OK, MessageBoxImage.Exclamation);
				}
				else
				{
					XElement f;
					foreach (XElement xelement2 in colFields)
					{
						f = xelement2;
						XElement newElement = new XElement(f.Name.LocalName, f.Value);
						f.Attributes().ToList<XAttribute>().ForEach(delegate(XAttribute attr)
						{
							newElement.Add(new XAttribute(attr.Name.LocalName, attr.Value));
						});
						newElement.Add(new XAttribute("text", TableColumnHelper.GetColumnText(tableName, f.Attribute("name").Value)));
						string text = (from col in TableColumnHelper.FindTableColumns(tableName).Elements("column")
							where col.Attribute("name").Value == f.Attribute("name").Value
							select col.Attribute("req").Value).ElementAtOrDefault<string>(0);
						newElement.Add(new XAttribute("req", text));
						xelement.Add(newElement);
					}
					this.columnView.DataContext = xelement;
				}
				IEnumerable<XElement> refFields = TableColumnHelper.GetRefFields(tableName);
				this.refView.DataContext = refFields;
				IEnumerable<XElement> langFields = TableColumnHelper.GetLangFields(tableName);
				this.langView.DataContext = langFields;
				XElement treeNode = TableColumnHelper.GetTreeNode(tableName);
				this.treeView.DataContext = treeNode;
				IEnumerable<XElement> sccfields = TableColumnHelper.GetSCCFields(tableName);
				this.statusView.DataContext = sccfields;
			}
		}

		// Token: 0x0600002E RID: 46 RVA: 0x000035FC File Offset: 0x000017FC
		private void Searc_PreviewKeyDown(object sender, KeyEventArgs e)
		{
			Key key = e.Key;
			if (key != Key.Return)
			{
				return;
			}
			this.Filter();
		}

		// Token: 0x0600002F RID: 47 RVA: 0x0000361B File Offset: 0x0000181B
		private void searchBox_TextChanged(object sender, TextChangedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x06000030 RID: 48 RVA: 0x00003623 File Offset: 0x00001823
		private void moduleCombo_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x06000031 RID: 49 RVA: 0x0000362B File Offset: 0x0000182B
		private void Button_Click(object sender, RoutedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x06000032 RID: 50 RVA: 0x00003750 File Offset: 0x00001950
		private void Filter()
		{
			string module = this.moduleCombo.SelectedItem as string;
			string[] keywords = this.searchBox.Text.Split(new char[] { '|' });
			this._filterView.Filter = delegate(object item)
			{
				bool? flag = null;
				string[] keywords2 = keywords;
				int i = 0;
				while (i < keywords2.Length)
				{
					string text = keywords2[i];
					string text2 = text;
					if (keywords.Count<string>() > 1)
					{
						text2 = text2.Trim();
					}
					XElement xelement = item as XElement;
					if (string.IsNullOrWhiteSpace(module) || !(xelement.Attribute("module").Value != module))
					{
						if (!string.IsNullOrEmpty(text2))
						{
							if (xelement.Attribute("name").Value.Contains(text2) || xelement.Attribute("desc").Value.Contains(text2))
							{
								return true;
							}
							flag = new bool?(new bool?(false) ?? false);
						}
						i++;
						continue;
					}
					return false;
				}
				return flag ?? true;
			};
			this.tableView.SelectedIndex = 0;
			this.statusItem.Content = string.Format("Table Count：{0}, Current：{1}", this.tableView.Items.Count, this.tableView.SelectedIndex + 1);
		}

		// Token: 0x06000033 RID: 51 RVA: 0x000037FC File Offset: 0x000019FC
		public void Dispose()
		{
			if (this._filterView != null)
			{
				this._filterView.Filter = null;
			}
			this.tableView.DataContext = null;
			this.moduleCombo.DataContext = null;
			this._tables.Clear();
			this._modules.Clear();
			this.tableView.SelectionChanged -= this.tableView_SelectionChanged;
			this.moduleCombo.PreviewKeyDown -= this.Searc_PreviewKeyDown;
			this.moduleCombo.SelectionChanged -= this.moduleCombo_SelectionChanged;
			this.searchBox.TextChanged -= this.searchBox_TextChanged;
		}

		// Token: 0x04000012 RID: 18
		private List<XElement> _tables = new List<XElement>();

		// Token: 0x04000013 RID: 19
		private List<string> _modules = new List<string>();

		// Token: 0x04000014 RID: 20
		private static ColumnViewerWindow _this;

		// Token: 0x04000015 RID: 21
		private ICollectionView _filterView;
	}
}
