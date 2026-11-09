using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Search.ViewModel;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using Xceed.Wpf.AvalonDock;
using Xceed.Wpf.AvalonDock.Layout;

namespace SpecDesigner.Search
{
	// Token: 0x02000005 RID: 5
	public partial class ReplaceBoxControl : UserControl, INotifyPropertyChanged, IDisposable
	{
		// Token: 0x17000007 RID: 7
		// (get) Token: 0x0600002C RID: 44 RVA: 0x00002B4E File Offset: 0x00000D4E
		public static string ContentID
		{
			get
			{
				return "ReplaceBox";
			}
		}

		// Token: 0x17000008 RID: 8
		// (get) Token: 0x0600002D RID: 45 RVA: 0x00002B55 File Offset: 0x00000D55
		// (set) Token: 0x0600002E RID: 46 RVA: 0x00002B5D File Offset: 0x00000D5D
		public bool IsMatchCase
		{
			get
			{
				return this._isMatchCase;
			}
			set
			{
				this._isMatchCase = value;
				this.OnPropertyChanged("IsMatchCase");
			}
		}

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x0600002F RID: 47 RVA: 0x00002B71 File Offset: 0x00000D71
		// (set) Token: 0x06000030 RID: 48 RVA: 0x00002B79 File Offset: 0x00000D79
		public bool IsEditableOnly
		{
			get
			{
				return this._isEditableOnly;
			}
			set
			{
				this._isEditableOnly = value;
			}
		}

		// Token: 0x1700000A RID: 10
		// (get) Token: 0x06000031 RID: 49 RVA: 0x00002B82 File Offset: 0x00000D82
		// (set) Token: 0x06000032 RID: 50 RVA: 0x00002B8A File Offset: 0x00000D8A
		public bool IsSelectedOnly
		{
			get
			{
				return this._isSelectedOnly;
			}
			set
			{
				this._isSelectedOnly = value;
			}
		}

		// Token: 0x1700000B RID: 11
		// (set) Token: 0x06000033 RID: 51 RVA: 0x00002B93 File Offset: 0x00000D93
		public string SearchKey
		{
			set
			{
				if (!string.IsNullOrWhiteSpace(value))
				{
					this.SearchCB.Text = value;
				}
			}
		}

		// Token: 0x06000034 RID: 52 RVA: 0x00002BAC File Offset: 0x00000DAC
		public ReplaceBoxControl()
		{
			this.InitializeComponent();
			this._model = new ReplaceBoxViewModel();
			base.DataContext = this._model;
			base.CommandBindings.Add(new CommandBinding(SearchCommands.NextCommnad, new ExecutedRoutedEventHandler(this.OnExecuteNext), new CanExecuteRoutedEventHandler(this.CanExecuteNext)));
			base.CommandBindings.Add(new CommandBinding(ReplaceCommands.ReplaceCommnad, new ExecutedRoutedEventHandler(this.OnExecuteReplace), new CanExecuteRoutedEventHandler(this.CanExecuteReplace)));
			base.CommandBindings.Add(new CommandBinding(ReplaceCommands.ReplaceAllCommnad, new ExecutedRoutedEventHandler(this.OnExecuteReplaceAll), new CanExecuteRoutedEventHandler(this.CanExecuteReplace)));
		}

		// Token: 0x06000035 RID: 53 RVA: 0x00002CB0 File Offset: 0x00000EB0
		private void SearchBoxControl_IsVisibleChanged(object sender, DependencyPropertyChangedEventArgs e)
		{
			if ((bool)e.NewValue)
			{
				base.Dispatcher.BeginInvoke(new Action(delegate
				{
					TextBox textBox = this.SearchCB.Template.FindName("PART_EditableTextBox", this.SearchCB) as TextBox;
					if (textBox != null)
					{
						textBox.Focus();
						textBox.SelectAll();
					}
				}), new object[0]);
				return;
			}
			Application.Current.MainWindow.Focus();
		}

		// Token: 0x06000036 RID: 54 RVA: 0x00002D01 File Offset: 0x00000F01
		public void Subscribe_TzpFileClosed(PackageKey key)
		{
			this._model.RemoveByProgramName(key);
		}

		// Token: 0x06000037 RID: 55 RVA: 0x00002D10 File Offset: 0x00000F10
		public void Subscribe_SearcyKeywordResult(SearchResultInfo args)
		{
			ReplaceResultInfo replaceResultInfo = new ReplaceResultInfo(args);
			replaceResultInfo.SearchInformation = args;
			replaceResultInfo.ReplaceAs = this.ReplaceAsCB.Text;
			this._model.AddSearchResult(replaceResultInfo);
		}

		// Token: 0x06000038 RID: 56 RVA: 0x00002D48 File Offset: 0x00000F48
		public static void Show(DockingManager dockManager, string key)
		{
			LayoutAnchorable layoutAnchorable = ReplaceBoxControl.Show(dockManager);
			ReplaceBoxControl replaceBoxControl = layoutAnchorable.Content as ReplaceBoxControl;
			if (replaceBoxControl != null)
			{
				replaceBoxControl.SearchKey = key;
			}
		}

		// Token: 0x06000039 RID: 57 RVA: 0x00002D84 File Offset: 0x00000F84
		public static LayoutAnchorable Show(DockingManager dockManager)
		{
			LayoutAnchorable layoutAnchorable = (from l in dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
				where l.ContentId == ReplaceBoxControl.ContentID
				select l).FirstOrDefault<LayoutAnchorable>();
			if (layoutAnchorable != null)
			{
				if (layoutAnchorable.Content == null)
				{
					layoutAnchorable.Content = new ReplaceBoxControl();
				}
				layoutAnchorable.IsVisible = true;
				layoutAnchorable.IsActive = true;
				layoutAnchorable.Hiding += ReplaceBoxControl.searchBoxWindow_Hiding;
				return layoutAnchorable;
			}
			LayoutAnchorable layoutAnchorable2 = new LayoutAnchorable
			{
				Title = (Application.Current.FindResource("WinTitle_SearchBoxTitle") as string),
				FloatingWidth = 250.0,
				FloatingHeight = 300.0,
				FloatingLeft = SystemParameters.PrimaryScreenWidth - 450.0,
				FloatingTop = SystemParameters.PrimaryScreenHeight / 2.0 - 300.0
			};
			layoutAnchorable2.AddToLayout(dockManager, AnchorableShowStrategy.Left);
			layoutAnchorable2.Content = ((ReplaceBoxControl._replaceBoxControl == null) ? new ReplaceBoxControl() : ReplaceBoxControl._replaceBoxControl);
			layoutAnchorable2.ContentId = ReplaceBoxControl.ContentID;
			layoutAnchorable2.Float();
			layoutAnchorable2.IsActive = true;
			layoutAnchorable2.Hiding += ReplaceBoxControl.searchBoxWindow_Hiding;
			return layoutAnchorable2;
		}

		// Token: 0x0600003A RID: 58 RVA: 0x00002EC1 File Offset: 0x000010C1
		private static void searchBoxWindow_Hiding(object sender, CancelEventArgs e)
		{
			(sender as LayoutAnchorable).Hiding -= ReplaceBoxControl.searchBoxWindow_Hiding;
			EventAggregatorManager.Global.GetEvent<ClosedSearchBoxEvent>().Publish(string.Empty);
		}

		// Token: 0x0600003B RID: 59 RVA: 0x00002EEE File Offset: 0x000010EE
		private void CanExecuteSearch(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = this.SearchCB.Text.Trim() != "";
		}

		// Token: 0x0600003C RID: 60 RVA: 0x00002F10 File Offset: 0x00001110
		private void OnExecuteSearch(object sender, ExecutedRoutedEventArgs e)
		{
			this.StartSearch();
		}

		// Token: 0x0600003D RID: 61 RVA: 0x00002F18 File Offset: 0x00001118
		private void CanExecutePrevious(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = this._model.CanMovePrevious;
		}

		// Token: 0x0600003E RID: 62 RVA: 0x00002F2B File Offset: 0x0000112B
		private void OnExecutePrevious(object sender, ExecutedRoutedEventArgs e)
		{
			this._model.MovePreviousResult();
		}

		// Token: 0x0600003F RID: 63 RVA: 0x00002F38 File Offset: 0x00001138
		private void CanExecuteNext(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000040 RID: 64 RVA: 0x00002F41 File Offset: 0x00001141
		private void OnExecuteNext(object sender, ExecutedRoutedEventArgs e)
		{
			this._model.MoveNextResult();
		}

		// Token: 0x06000041 RID: 65 RVA: 0x00002F4E File Offset: 0x0000114E
		private void CanExecuteReplace(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000042 RID: 66 RVA: 0x00002F58 File Offset: 0x00001158
		private void OnExecuteReplace(object sender, ExecutedRoutedEventArgs e)
		{
			ReplaceResultInfo replaceResultInfo = new ReplaceResultInfo();
			replaceResultInfo.ReplaceAs = this.ReplaceAsCB.Text;
			replaceResultInfo.Keyword = this.SearchCB.Text;
			replaceResultInfo.SelectedOnly = this.IsSelectedOnly;
			replaceResultInfo.IsMatchCase = this.IsMatchCase;
			this._model.PublishReplaceEvent(replaceResultInfo);
		}

		// Token: 0x06000043 RID: 67 RVA: 0x00002FB4 File Offset: 0x000011B4
		private void OnExecuteReplaceAll(object sender, ExecutedRoutedEventArgs e)
		{
			ReplaceResultInfo replaceResultInfo = new ReplaceResultInfo();
			replaceResultInfo.ReplaceAs = this.ReplaceAsCB.Text;
			replaceResultInfo.Keyword = this.SearchCB.Text;
			replaceResultInfo.SelectedOnly = this.IsSelectedOnly;
			replaceResultInfo.IsMatchCase = this.IsMatchCase;
			this._model.RublishReplaceAllEvent(replaceResultInfo);
		}

		// Token: 0x06000044 RID: 68 RVA: 0x00003010 File Offset: 0x00001210
		private void StartSearch()
		{
			string text = (this.SearchTargetCB.SelectedItem as ComboBoxItem).Tag.ToString();
			this._model.Search(this.SearchCB.Text, this.IsMatchCase, this.IsEditableOnly, this.IsSelectedOnly, text);
		}

		// Token: 0x06000045 RID: 69 RVA: 0x00003061 File Offset: 0x00001261
		private void DataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			this._model.PublishSelectedEvent();
		}

		// Token: 0x14000002 RID: 2
		// (add) Token: 0x06000046 RID: 70 RVA: 0x00003070 File Offset: 0x00001270
		// (remove) Token: 0x06000047 RID: 71 RVA: 0x000030A8 File Offset: 0x000012A8
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000048 RID: 72 RVA: 0x000030DD File Offset: 0x000012DD
		private void OnPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x06000049 RID: 73 RVA: 0x000030FC File Offset: 0x000012FC
		public void Dispose()
		{
			EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Unsubscribe(new Action<SearchResultInfo>(this.Subscribe_SearcyKeywordResult));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.Subscribe_TzpFileClosed));
			base.IsVisibleChanged -= this.SearchBoxControl_IsVisibleChanged;
		}

		// Token: 0x0600004A RID: 74 RVA: 0x00003151 File Offset: 0x00001351
		private CollectionView GetDefaultView()
		{
			return (CollectionView)CollectionViewSource.GetDefaultView(this._model.SearchResultList);
		}

		// Token: 0x0600004B RID: 75 RVA: 0x00003168 File Offset: 0x00001368
		private void dataGrid_Sorting(object sender, DataGridSortingEventArgs e)
		{
			ListSortDirection listSortDirection = ((e.Column.SortDirection != ListSortDirection.Ascending) ? ListSortDirection.Ascending : ListSortDirection.Descending);
			e.Column.SortDirection = new ListSortDirection?(listSortDirection);
			e.Handled = true;
			string sortMemberPath = e.Column.SortMemberPath;
			string text;
			if ((text = sortMemberPath) != null)
			{
				if (text == "Match")
				{
					this.GetDefaultView().SortDescriptions.Clear();
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("Memo", listSortDirection));
					return;
				}
				if (text == "Program")
				{
					this.GetDefaultView().SortDescriptions.Clear();
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("ProgramKey.Program", listSortDirection));
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("Memo", ListSortDirection.Ascending));
					return;
				}
				if (text == "SourceType")
				{
					this.GetDefaultView().SortDescriptions.Clear();
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("SourceType", listSortDirection));
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("ProgramKey.Program", ListSortDirection.Ascending));
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("Memo", ListSortDirection.Ascending));
					return;
				}
				if (text == "IsEditable")
				{
					this.GetDefaultView().SortDescriptions.Clear();
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("IsEditable", listSortDirection));
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("ProgramKey.Program", ListSortDirection.Ascending));
					this.GetDefaultView().SortDescriptions.Add(new SortDescription("Memo", ListSortDirection.Ascending));
					return;
				}
			}
			e.Handled = false;
		}

		// Token: 0x0600004C RID: 76 RVA: 0x0000336C File Offset: 0x0000156C
		private void SearchFilter(string condition)
		{
			if (this._searchCustomerView == null)
			{
				this._searchCustomerView = CollectionViewSource.GetDefaultView(this.SearchCB.ItemsSource);
			}
			this._searchCustomerView.Filter = (object item) => item.ToString().IndexOf(condition, StringComparison.CurrentCultureIgnoreCase) != -1;
		}

		// Token: 0x0600004D RID: 77 RVA: 0x000033E0 File Offset: 0x000015E0
		private void ReplaceAsFilter(string condition)
		{
			if (this._replaceAsCustomerView == null)
			{
				this._replaceAsCustomerView = CollectionViewSource.GetDefaultView(this.ReplaceAsCB.ItemsSource);
			}
			this._replaceAsCustomerView.Filter = (object item) => item.ToString().IndexOf(condition, StringComparison.CurrentCultureIgnoreCase) != -1;
		}

		// Token: 0x0600004E RID: 78 RVA: 0x00003430 File Offset: 0x00001630
		private void SearchCB_KeyUp(object sender, KeyEventArgs e)
		{
			Key key = e.Key;
			if (key == Key.Return)
			{
				ReplaceCommands.ReplaceCommnad.Execute(null, this);
				return;
			}
			if (key == Key.Escape)
			{
				this.SearchCB.IsDropDownOpen = false;
				return;
			}
			if (this.SearchCB.Text.Length == 0 && this._searchCustomerView != null)
			{
				if (this._searchCustomerView != null)
				{
					this._searchCustomerView.Filter = null;
				}
				this.SearchCB.IsDropDownOpen = false;
				return;
			}
			this.SearchFilter(this.SearchCB.Text);
			this.SearchCB.IsDropDownOpen = this.SearchCB.Items.Count > 0;
		}

		// Token: 0x0600004F RID: 79 RVA: 0x000034D8 File Offset: 0x000016D8
		private void ReplaceAsCB_KeyUp(object sender, KeyEventArgs e)
		{
			Key key = e.Key;
			if (key == Key.Return)
			{
				ReplaceCommands.ReplaceCommnad.Execute(null, this);
				return;
			}
			if (key == Key.Escape)
			{
				this.ReplaceAsCB.IsDropDownOpen = false;
				return;
			}
			if (this.ReplaceAsCB.Text.Length == 0)
			{
				if (this._replaceAsCustomerView != null)
				{
					this._replaceAsCustomerView.Filter = null;
				}
				this.ReplaceAsCB.IsDropDownOpen = false;
				return;
			}
			this.ReplaceAsFilter(this.ReplaceAsCB.Text);
			this.ReplaceAsCB.IsDropDownOpen = this.ReplaceAsCB.Items.Count > 0;
		}

		// Token: 0x04000011 RID: 17
		private static ReplaceBoxControl _replaceBoxControl;

		// Token: 0x04000012 RID: 18
		private ReplaceBoxViewModel _model;

		// Token: 0x04000013 RID: 19
		private bool _isMatchCase = true;

		// Token: 0x04000014 RID: 20
		private bool _isEditableOnly;

		// Token: 0x04000015 RID: 21
		private bool _isSelectedOnly;

		// Token: 0x04000017 RID: 23
		private ICollectionView _searchCustomerView;

		// Token: 0x04000018 RID: 24
		private ICollectionView _replaceAsCustomerView;
	}
}
