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
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.TableViewer
{
	// Token: 0x02000002 RID: 2
	public partial class TableViewerWindow : Window, IDisposable
	{
		// Token: 0x17000001 RID: 1
		// (get) Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public static TableViewerWindow This
		{
			get
			{
				if (TableViewerWindow._this == null)
				{
					TableViewerWindow._this = new TableViewerWindow();
				}
				return TableViewerWindow._this;
			}
		}

		// Token: 0x06000002 RID: 2 RVA: 0x00002068 File Offset: 0x00000268
		private TableViewerWindow()
		{
			this.InitializeComponent();
			base.Title = Application.Current.FindResource("tableView_WindowTitle") as string;
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

		// Token: 0x06000003 RID: 3 RVA: 0x00002184 File Offset: 0x00000384
		private void BindCommands()
		{
			base.CommandBindings.Add(new CommandBinding(TableColumnCommands.CopyAllColumnsNameCommand, new ExecutedRoutedEventHandler(TableColumnCommands.ExecutedCopyAllColumnsName), new CanExecuteRoutedEventHandler(TableColumnCommands.CanExecuteCopyAllColumnsName)));
			base.CommandBindings.Add(new CommandBinding(TableColumnCommands.CopyAllColumnsToRecordCommand, new ExecutedRoutedEventHandler(TableColumnCommands.ExecutedCopyAllColumnsToRecord), new CanExecuteRoutedEventHandler(TableColumnCommands.CanExecuteCopyAllColumnsToRecord)));
		}

		// Token: 0x06000004 RID: 4 RVA: 0x000021ED File Offset: 0x000003ED
		private void Current_Exit(object sender, ExitEventArgs e)
		{
			Application.Current.Exit -= this.Current_Exit;
			EventAggregatorManager.Global.GetEvent<LoadSpecReferFilesEvent>().Unsubscribe(new Action<string>(this.Reload));
			base.Close();
		}

		// Token: 0x06000005 RID: 5 RVA: 0x00002228 File Offset: 0x00000428
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

		// Token: 0x06000006 RID: 6 RVA: 0x00002298 File Offset: 0x00000498
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

		// Token: 0x06000007 RID: 7 RVA: 0x0000239C File Offset: 0x0000059C
		public new void Show()
		{
			this.LoadTableSchema();
			PackageKey packageKey = Application.Current.MainWindow.Tag as PackageKey;
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

		// Token: 0x06000008 RID: 8 RVA: 0x0000248E File Offset: 0x0000068E
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
			this._tables.Clear();
			this._modules.Clear();
		}

		// Token: 0x06000009 RID: 9 RVA: 0x000024B4 File Offset: 0x000006B4
		private void tableView_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			XElement xelement = this.tableView.SelectedItem as XElement;
			if (xelement == null)
			{
				this.columnView.DataContext = null;
				this.tableLabel.Text = string.Empty;
			}
			else
			{
				XElement xelement2 = TableColumnHelper.FindTableColumns(xelement.Attribute("name").Value);
				this.columnView.DataContext = xelement2;
				this.tableLabel.Text = string.Format(base.FindResource("tableView_ColumnTitle") as string, xelement.Attribute("name").Value, xelement.Attribute("desc").Value);
			}
			this.statusItem.Content = string.Format("Table Count: {0}, Selected: {1}", this.tableView.Items.Count, this.tableView.SelectedIndex + 1);
		}

		// Token: 0x0600000A RID: 10 RVA: 0x000025A0 File Offset: 0x000007A0
		private void Searc_PreviewKeyDown(object sender, KeyEventArgs e)
		{
			Key key = e.Key;
			if (key != Key.Return)
			{
				return;
			}
			this.Filter();
		}

		// Token: 0x0600000B RID: 11 RVA: 0x000025BF File Offset: 0x000007BF
		private void searchBox_TextChanged(object sender, TextChangedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x0600000C RID: 12 RVA: 0x000025C7 File Offset: 0x000007C7
		private void moduleCombo_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x0600000D RID: 13 RVA: 0x000025CF File Offset: 0x000007CF
		private void Button_Click(object sender, RoutedEventArgs e)
		{
			this.Filter();
		}

		// Token: 0x0600000E RID: 14 RVA: 0x000026F4 File Offset: 0x000008F4
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

		// Token: 0x0600000F RID: 15 RVA: 0x000027A0 File Offset: 0x000009A0
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

		// Token: 0x04000001 RID: 1
		private List<XElement> _tables = new List<XElement>();

		// Token: 0x04000002 RID: 2
		private List<string> _modules = new List<string>();

		// Token: 0x04000003 RID: 3
		private static TableViewerWindow _this;

		// Token: 0x04000004 RID: 4
		private ICollectionView _filterView;
	}
}
