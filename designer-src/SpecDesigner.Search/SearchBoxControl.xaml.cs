using System;
using System.CodeDom.Compiler;
using System.Collections.Specialized;
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
	// Token: 0x02000002 RID: 2
	public partial class SearchBoxControl : UserControl, INotifyPropertyChanged
	{
		// Token: 0x17000001 RID: 1
		// (get) Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public static string ContentID
		{
			get
			{
				return "SearchBox";
			}
		}

		// Token: 0x17000002 RID: 2
		// (get) Token: 0x06000002 RID: 2 RVA: 0x00002057 File Offset: 0x00000257
		// (set) Token: 0x06000003 RID: 3 RVA: 0x0000205F File Offset: 0x0000025F
		public bool IsSearchSpec
		{
			get
			{
				return this._isSearchSpec;
			}
			set
			{
				this._isSearchSpec = value;
				this.OnPropertyChanged("IsSearchSpec");
			}
		}

		// Token: 0x17000003 RID: 3
		// (get) Token: 0x06000004 RID: 4 RVA: 0x00002073 File Offset: 0x00000273
		// (set) Token: 0x06000005 RID: 5 RVA: 0x0000207B File Offset: 0x0000027B
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

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x06000006 RID: 6 RVA: 0x0000208F File Offset: 0x0000028F
		// (set) Token: 0x06000007 RID: 7 RVA: 0x00002097 File Offset: 0x00000297
		public bool IsRegExMode
		{
			get
			{
				return this._isRegExMode;
			}
			set
			{
				this._isRegExMode = value;
				this.OnPropertyChanged("IsRegExMode");
			}
		}

		// Token: 0x06000008 RID: 8 RVA: 0x000020AC File Offset: 0x000002AC
		public SearchBoxControl()
		{
			this.InitializeComponent();
			this._model = new SearchBoxViewModel(this);
			base.DataContext = this._model;
			this.dataGrid.ItemsSource = this.GetDefaultView();
			EventAggregatorManager.Global.GetEvent<SearchKeywordResultEvent>().Subscribe(new Action<SearchResultInfo>(this.Subscribe_SearcyKeywordResult));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.Subscribe_TzpFileClosed));
			base.IsVisibleChanged += this.SearchBoxControl_IsVisibleChanged;
			base.CommandBindings.Add(new CommandBinding(SearchCommands.SearchCommnad, new ExecutedRoutedEventHandler(this.OnExecuteSearch), new CanExecuteRoutedEventHandler(this.CanExecuteSearch)));
			base.CommandBindings.Add(new CommandBinding(SearchCommands.PreviousCommnad, new ExecutedRoutedEventHandler(this.OnExecutePrevious), new CanExecuteRoutedEventHandler(this.CanExecutePrevious)));
			base.CommandBindings.Add(new CommandBinding(SearchCommands.NextCommnad, new ExecutedRoutedEventHandler(this.OnExecuteNext), new CanExecuteRoutedEventHandler(this.CanExecuteNext)));
		}

		// Token: 0x06000009 RID: 9 RVA: 0x000021C9 File Offset: 0x000003C9
		private void CanExecuteSearch(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = this.SearchCB.Text.Trim() != "";
		}

		// Token: 0x0600000A RID: 10 RVA: 0x000021EB File Offset: 0x000003EB
		private void OnExecuteSearch(object sender, ExecutedRoutedEventArgs e)
		{
			this.beginSearch(false);
			this.ComparingResultList();
		}

		// Token: 0x0600000B RID: 11 RVA: 0x000021FC File Offset: 0x000003FC
		private void ComparingResultList()
		{
			for (int i = 0; i < this._model.SearchResultList.Count<SearchResultInfo>(); i++)
			{
				int memo = this._model.SearchResultList[i].Memo;
				string match = this._model.SearchResultList[i].Match;
				bool flag = false;
				for (int j = 0; j < this._model.SearchResultList.Count<SearchResultInfo>(); j++)
				{
					MemoType memoType;
					if (this._model.SearchResultList[i].ProgramKey.Memo == MemoType.CodeDiff)
					{
						memoType = MemoType.None;
					}
					else
					{
						memoType = MemoType.CodeDiff;
					}
					if (this._model.SearchResultList[j].Memo == memo && this._model.SearchResultList[j].Match == match && this._model.SearchResultList[j].ProgramKey.Memo == memoType)
					{
						this._model.SearchResultList[i].ComparingSame = true;
						this._model.SearchResultList[j].ComparingSame = true;
						flag = true;
						break;
					}
				}
				if (!flag)
				{
					this._model.SearchResultList[i].ComparingSame = false;
				}
			}
		}

		// Token: 0x0600000C RID: 12 RVA: 0x0000234B File Offset: 0x0000054B
		private void CanExecutePrevious(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = this._model.CanMovePrevious;
		}

		// Token: 0x0600000D RID: 13 RVA: 0x0000235E File Offset: 0x0000055E
		private void OnExecutePrevious(object sender, ExecutedRoutedEventArgs e)
		{
			this._model.MovePreviousResult();
			this.dataGrid.ScrollIntoView(this.dataGrid.SelectedItem, this.dataGrid.Columns[0]);
		}

		// Token: 0x0600000E RID: 14 RVA: 0x00002392 File Offset: 0x00000592
		private void CanExecuteNext(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = this._model.CanMoveNext;
		}

		// Token: 0x0600000F RID: 15 RVA: 0x000023A5 File Offset: 0x000005A5
		private void OnExecuteNext(object sender, ExecutedRoutedEventArgs e)
		{
			this._model.MoveNextResult();
			this.dataGrid.ScrollIntoView(this.dataGrid.SelectedItem, this.dataGrid.Columns[0]);
		}

		// Token: 0x06000010 RID: 16 RVA: 0x0000241C File Offset: 0x0000061C
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

		// Token: 0x06000011 RID: 17 RVA: 0x0000246D File Offset: 0x0000066D
		public void Subscribe_TzpFileClosed(PackageKey key)
		{
			this._model.RemoveByProgramName(key);
		}

		// Token: 0x06000012 RID: 18 RVA: 0x0000247B File Offset: 0x0000067B
		public void Subscribe_SearcyKeywordResult(SearchResultInfo info)
		{
			this._model.AddSearchResult(info);
		}

		// Token: 0x17000005 RID: 5
		// (set) Token: 0x06000013 RID: 19 RVA: 0x00002489 File Offset: 0x00000689
		public string SearchKey
		{
			set
			{
				if (!string.IsNullOrWhiteSpace(value))
				{
					this.SearchCB.Text = value;
					this.SearchingKey = value;
				}
			}
		}

		// Token: 0x06000014 RID: 20 RVA: 0x000024A8 File Offset: 0x000006A8
		public static void Show(DockingManager dockManager, string key)
		{
			LayoutAnchorable layoutAnchorable = SearchBoxControl.Show(dockManager, true);
			SearchBoxControl searchBoxControl = layoutAnchorable.Content as SearchBoxControl;
			if (searchBoxControl != null)
			{
				searchBoxControl.SearchKey = key;
			}
		}

		// Token: 0x06000015 RID: 21 RVA: 0x000024D3 File Offset: 0x000006D3
		public static void Show(DockingManager dockManager)
		{
			SearchBoxControl.Show(dockManager, true);
		}

		// Token: 0x06000016 RID: 22 RVA: 0x000024F0 File Offset: 0x000006F0
		public static LayoutAnchorable Show(DockingManager dockManager, bool isActive)
		{
			LayoutAnchorable layoutAnchorable = (from l in dockManager.Layout.Descendents().OfType<LayoutAnchorable>()
				where l.ContentId == SearchBoxControl.ContentID
				select l).FirstOrDefault<LayoutAnchorable>();
			if (layoutAnchorable != null)
			{
				if (layoutAnchorable.Content == null)
				{
					layoutAnchorable.Content = new SearchBoxControl();
				}
				layoutAnchorable.IsVisible = isActive;
				layoutAnchorable.IsActive = isActive;
				layoutAnchorable.Hiding += SearchBoxControl.searchBoxWindow_Hiding;
				return layoutAnchorable;
			}
			LayoutAnchorable layoutAnchorable2 = new LayoutAnchorable
			{
				Title = (Application.Current.FindResource("WinTitle_SearchBoxTitle") as string),
				FloatingWidth = 250.0,
				FloatingHeight = 400.0,
				FloatingLeft = SystemParameters.PrimaryScreenWidth - 450.0,
				FloatingTop = SystemParameters.PrimaryScreenHeight / 2.0 - 300.0
			};
			layoutAnchorable2.AddToLayout(dockManager, AnchorableShowStrategy.Left);
			layoutAnchorable2.Content = ((SearchBoxControl._searchBoxControl == null) ? new SearchBoxControl() : SearchBoxControl._searchBoxControl);
			layoutAnchorable2.ContentId = SearchBoxControl.ContentID;
			layoutAnchorable2.Float();
			layoutAnchorable2.IsActive = isActive;
			layoutAnchorable2.Hiding += SearchBoxControl.searchBoxWindow_Hiding;
			return layoutAnchorable2;
		}

		// Token: 0x06000017 RID: 23 RVA: 0x0000262D File Offset: 0x0000082D
		private static void searchBox_Hiding(object sender, CancelEventArgs e)
		{
			(sender as LayoutAnchorable).Hiding -= SearchBoxControl.searchBoxWindow_Hiding;
			EventAggregatorManager.Global.GetEvent<ClosedSearchBoxEvent>().Publish(string.Empty);
		}

		// Token: 0x06000018 RID: 24 RVA: 0x0000265A File Offset: 0x0000085A
		private static void searchBoxWindow_Hiding(object sender, CancelEventArgs e)
		{
			(sender as LayoutAnchorable).Hiding -= SearchBoxControl.searchBoxWindow_Hiding;
			EventAggregatorManager.Global.GetEvent<ClosedSearchBoxEvent>().Publish(string.Empty);
		}

		// Token: 0x06000019 RID: 25 RVA: 0x00002688 File Offset: 0x00000888
		private void beginSearch(bool isAutoSelect)
		{
			this._model.ClearSearchResult();
			this._model.SearchResultList.CollectionChanged -= this.SearchResultList_CollectionChanged;
			if (isAutoSelect)
			{
				this._model.SearchResultList.CollectionChanged += this.SearchResultList_CollectionChanged;
			}
			this.SearchingKey = this.SearchCB.Text;
			this._model.AddKeyWord(this.SearchingKey);
			string text = (this.SearchTargetCB.SelectedItem as ComboBoxItem).Tag.ToString();
			PackageKey packageKey = Application.Current.MainWindow.Tag as PackageKey;
			if (text.Equals("All Open Documents"))
			{
				packageKey = null;
			}
			SearchKeywordEventArgs e = new SearchKeywordEventArgs(packageKey, this.SearchingKey, this.IsMatchCase)
			{
				IsRegExMode = this.IsRegExMode
			};
			e.IsSearchSpec = this.IsSearchSpec;
			EventAggregatorManager.Global.GetEvent<SearchKeywordEvent>().Publish(e);
		}

		// Token: 0x0600001A RID: 26 RVA: 0x00002779 File Offset: 0x00000979
		private void SearchResultList_CollectionChanged(object sender, NotifyCollectionChangedEventArgs e)
		{
			this._model.PublishSelectedEvent(0);
		}

		// Token: 0x0600001B RID: 27 RVA: 0x00002787 File Offset: 0x00000987
		private void DataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			this._model.PublishSelectedEvent();
		}

		// Token: 0x14000001 RID: 1
		// (add) Token: 0x0600001C RID: 28 RVA: 0x00002794 File Offset: 0x00000994
		// (remove) Token: 0x0600001D RID: 29 RVA: 0x000027CC File Offset: 0x000009CC
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600001E RID: 30 RVA: 0x00002801 File Offset: 0x00000A01
		public void OnPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x0600001F RID: 31 RVA: 0x0000281D File Offset: 0x00000A1D
		private CollectionView GetDefaultView()
		{
			return (CollectionView)CollectionViewSource.GetDefaultView(this._model.SearchResultList);
		}

		// Token: 0x06000020 RID: 32 RVA: 0x00002834 File Offset: 0x00000A34
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
			}
			e.Handled = false;
		}

		// Token: 0x04000001 RID: 1
		private static SearchBoxControl _searchBoxControl;

		// Token: 0x04000002 RID: 2
		private SearchBoxViewModel _model;

		// Token: 0x04000003 RID: 3
		private string SearchingKey;

		// Token: 0x04000004 RID: 4
		private bool _isSearchSpec;

		// Token: 0x04000005 RID: 5
		private bool _isMatchCase = true;

		// Token: 0x04000006 RID: 6
		private bool _isRegExMode;
	}
}
