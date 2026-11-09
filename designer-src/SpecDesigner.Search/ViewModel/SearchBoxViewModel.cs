using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Search.ViewModel
{
	// Token: 0x02000006 RID: 6
	public class SearchBoxViewModel : INotifyPropertyChanged
	{
		// Token: 0x06000055 RID: 85 RVA: 0x00003630 File Offset: 0x00001830
		public SearchBoxViewModel(SearchBoxControl searchBoxControl)
		{
			this.searchBoxControl = searchBoxControl;
		}

		// Token: 0x06000056 RID: 86 RVA: 0x00003655 File Offset: 0x00001855
		public SearchBoxViewModel()
		{
		}

		// Token: 0x1700000C RID: 12
		// (get) Token: 0x06000057 RID: 87 RVA: 0x00003673 File Offset: 0x00001873
		public string Count
		{
			get
			{
				if (this._searchHistory.Count != 0)
				{
					return string.Format("{0} hits", this._searchResultList.Count);
				}
				return string.Empty;
			}
		}

		// Token: 0x1700000D RID: 13
		// (get) Token: 0x06000058 RID: 88 RVA: 0x000036A2 File Offset: 0x000018A2
		// (set) Token: 0x06000059 RID: 89 RVA: 0x000036AA File Offset: 0x000018AA
		public ObservableCollection<string> SearchHistory
		{
			get
			{
				return this._searchHistory;
			}
			set
			{
				this._searchHistory = value;
				this.NotifyPropertyChanged("SearchHistory");
			}
		}

		// Token: 0x1700000E RID: 14
		// (get) Token: 0x0600005A RID: 90 RVA: 0x000036BE File Offset: 0x000018BE
		// (set) Token: 0x0600005B RID: 91 RVA: 0x000036C6 File Offset: 0x000018C6
		public ObservableCollection<SearchResultInfo> SearchResultList
		{
			get
			{
				return this._searchResultList;
			}
			set
			{
				this._searchResultList = value;
				this.NotifyPropertyChanged("SearchResultList");
			}
		}

		// Token: 0x14000003 RID: 3
		// (add) Token: 0x0600005C RID: 92 RVA: 0x000036DC File Offset: 0x000018DC
		// (remove) Token: 0x0600005D RID: 93 RVA: 0x00003714 File Offset: 0x00001914
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x0600005E RID: 94 RVA: 0x00003749 File Offset: 0x00001949
		protected void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x1700000F RID: 15
		// (get) Token: 0x0600005F RID: 95 RVA: 0x00003765 File Offset: 0x00001965
		// (set) Token: 0x06000060 RID: 96 RVA: 0x0000376D File Offset: 0x0000196D
		public SearchResultInfo CurrentSearchResult
		{
			get
			{
				return this._currentSearchResult;
			}
			set
			{
				this._currentSearchResult = value;
				this.NotifyPropertyChanged("CurrentSearchResult");
			}
		}

		// Token: 0x06000061 RID: 97 RVA: 0x00003781 File Offset: 0x00001981
		internal void AddKeyWord(string searchKey)
		{
			if (!this.SearchHistory.Contains(searchKey))
			{
				this.SearchHistory.Add(searchKey);
				this._searchResultList.Clear();
			}
		}

		// Token: 0x06000062 RID: 98 RVA: 0x000037A8 File Offset: 0x000019A8
		internal void AddSearchResult(SearchResultInfo info)
		{
			this._searchResultList.Add(info);
			this.CurrentSearchResult = this._searchResultList[0];
			this.NotifyPropertyChanged("Count");
		}

		// Token: 0x17000010 RID: 16
		// (get) Token: 0x06000063 RID: 99 RVA: 0x000037D3 File Offset: 0x000019D3
		public bool CanMoveNext
		{
			get
			{
				return this._searchResultList.Count<SearchResultInfo>() != 0 && this._searchResultList.IndexOf(this.CurrentSearchResult) != this._searchResultList.Count - 1;
			}
		}

		// Token: 0x06000064 RID: 100 RVA: 0x00003808 File Offset: 0x00001A08
		internal void MoveNextResult()
		{
			if (this.CurrentSearchResult == null && this._searchResultList.Count<SearchResultInfo>() == 0)
			{
				return;
			}
			if (this.CurrentSearchResult == null)
			{
				this.CurrentSearchResult = this._searchResultList[0];
			}
			else
			{
				int num = this._searchResultList.IndexOf(this.CurrentSearchResult);
				if (num < this._searchResultList.Count - 1)
				{
					this.CurrentSearchResult = this._searchResultList[num + 1];
				}
			}
			this.NotifyPropertyChanged("CurrentSearchResult");
			this.PublishSelectedEvent();
		}

		// Token: 0x17000011 RID: 17
		// (get) Token: 0x06000065 RID: 101 RVA: 0x0000388E File Offset: 0x00001A8E
		public bool CanMovePrevious
		{
			get
			{
				return this._searchResultList.Count<SearchResultInfo>() != 0 && this._searchResultList.IndexOf(this.CurrentSearchResult) != 0;
			}
		}

		// Token: 0x06000066 RID: 102 RVA: 0x000038B8 File Offset: 0x00001AB8
		internal void MovePreviousResult()
		{
			if (this.CurrentSearchResult == null && this._searchResultList.Count<SearchResultInfo>() == 0)
			{
				return;
			}
			if (this.CurrentSearchResult == null)
			{
				this.CurrentSearchResult = this._searchResultList[0];
			}
			else
			{
				int num = this._searchResultList.IndexOf(this.CurrentSearchResult);
				if (num > 0)
				{
					this.CurrentSearchResult = this._searchResultList[num - 1];
				}
			}
			this.NotifyPropertyChanged("CurrentSearchResult");
			this.PublishSelectedEvent();
		}

		// Token: 0x06000067 RID: 103 RVA: 0x00003932 File Offset: 0x00001B32
		internal void ClearSearchResult()
		{
			this._searchResultList.Clear();
			this.CurrentSearchResult = null;
			this.NotifyPropertyChanged("CurrentSearchResult");
			this.NotifyPropertyChanged("Count");
		}

		// Token: 0x06000068 RID: 104 RVA: 0x0000395C File Offset: 0x00001B5C
		public void PublishSelectedEvent()
		{
			if (this.CurrentSearchResult != null)
			{
				EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Publish(this.CurrentSearchResult);
			}
		}

		// Token: 0x06000069 RID: 105 RVA: 0x0000397B File Offset: 0x00001B7B
		public void PublishSelectedEvent(int index)
		{
			if (this.SearchResultList.Count > index)
			{
				this.CurrentSearchResult = this.SearchResultList[index];
				this.PublishSelectedEvent();
			}
		}

		// Token: 0x0600006A RID: 106 RVA: 0x000039C0 File Offset: 0x00001BC0
		public void RemoveByProgramName(PackageKey key)
		{
			List<SearchResultInfo> list = this._searchResultList.Where<SearchResultInfo>((SearchResultInfo info) => info.ProgramKey == key).ToList<SearchResultInfo>();
			foreach (SearchResultInfo searchResultInfo in list)
			{
				this._searchResultList.Remove(searchResultInfo);
				this.NotifyPropertyChanged("Count");
			}
		}

		// Token: 0x04000021 RID: 33
		private SearchBoxControl searchBoxControl;

		// Token: 0x04000022 RID: 34
		private ObservableCollection<string> _searchHistory = new ObservableCollection<string>();

		// Token: 0x04000023 RID: 35
		private ObservableCollection<SearchResultInfo> _searchResultList = new ObservableCollection<SearchResultInfo>();

		// Token: 0x04000025 RID: 37
		private SearchResultInfo _currentSearchResult;
	}
}
