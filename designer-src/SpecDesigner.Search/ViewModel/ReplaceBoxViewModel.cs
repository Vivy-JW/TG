using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Windows;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Search.ViewModel
{
	// Token: 0x02000007 RID: 7
	public class ReplaceBoxViewModel : INotifyPropertyChanged
	{
		// Token: 0x17000012 RID: 18
		// (get) Token: 0x0600006C RID: 108 RVA: 0x00003A75 File Offset: 0x00001C75
		// (set) Token: 0x0600006D RID: 109 RVA: 0x00003A7D File Offset: 0x00001C7D
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

		// Token: 0x17000013 RID: 19
		// (get) Token: 0x0600006E RID: 110 RVA: 0x00003A91 File Offset: 0x00001C91
		// (set) Token: 0x0600006F RID: 111 RVA: 0x00003A99 File Offset: 0x00001C99
		public ObservableCollection<ReplaceResultInfo> SearchResultList
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

		// Token: 0x14000004 RID: 4
		// (add) Token: 0x06000070 RID: 112 RVA: 0x00003AB0 File Offset: 0x00001CB0
		// (remove) Token: 0x06000071 RID: 113 RVA: 0x00003AE8 File Offset: 0x00001CE8
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000072 RID: 114 RVA: 0x00003B1D File Offset: 0x00001D1D
		protected void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x17000014 RID: 20
		// (get) Token: 0x06000073 RID: 115 RVA: 0x00003B39 File Offset: 0x00001D39
		// (set) Token: 0x06000074 RID: 116 RVA: 0x00003B41 File Offset: 0x00001D41
		public string ReplaceAs
		{
			get
			{
				return this._replaceAs;
			}
			set
			{
				this._replaceAs = value;
				this.AddReplaceAs(this._replaceAs);
			}
		}

		// Token: 0x17000015 RID: 21
		// (get) Token: 0x06000075 RID: 117 RVA: 0x00003B56 File Offset: 0x00001D56
		// (set) Token: 0x06000076 RID: 118 RVA: 0x00003B5E File Offset: 0x00001D5E
		public ObservableCollection<string> ReplaceHistory
		{
			get
			{
				return this._replaceHistory;
			}
			set
			{
				this._replaceHistory = value;
				this.NotifyPropertyChanged("ReplaceHistory");
			}
		}

		// Token: 0x17000016 RID: 22
		// (get) Token: 0x06000077 RID: 119 RVA: 0x00003B72 File Offset: 0x00001D72
		// (set) Token: 0x06000078 RID: 120 RVA: 0x00003B7A File Offset: 0x00001D7A
		public ReplaceResultInfo CurrentSearchResult
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

		// Token: 0x06000079 RID: 121 RVA: 0x00003B8E File Offset: 0x00001D8E
		private void AddKeyWord(string searchKey)
		{
			if (!this.SearchHistory.Contains(searchKey))
			{
				this.SearchHistory.Add(searchKey);
				this._searchResultList.Clear();
			}
		}

		// Token: 0x0600007A RID: 122 RVA: 0x00003BB5 File Offset: 0x00001DB5
		private void AddReplaceAs(string replaceKey)
		{
			if (!this.ReplaceHistory.Contains(replaceKey))
			{
				this.ReplaceHistory.Add(replaceKey);
			}
		}

		// Token: 0x0600007B RID: 123 RVA: 0x00003BD1 File Offset: 0x00001DD1
		internal void AddSearchResult(ReplaceResultInfo info)
		{
			this._searchResultList.Add(info);
			this.CurrentSearchResult = this._searchResultList[0];
		}

		// Token: 0x17000017 RID: 23
		// (get) Token: 0x0600007C RID: 124 RVA: 0x00003BF1 File Offset: 0x00001DF1
		public bool CanMoveNext
		{
			get
			{
				return this._searchResultList.Count<ReplaceResultInfo>() != 0 && this._searchResultList.IndexOf(this.CurrentSearchResult) != this._searchResultList.Count - 1;
			}
		}

		// Token: 0x0600007D RID: 125 RVA: 0x00003C28 File Offset: 0x00001E28
		internal void MoveNextResult()
		{
			if (this.CurrentSearchResult == null && this._searchResultList.Count<ReplaceResultInfo>() == 0)
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

		// Token: 0x17000018 RID: 24
		// (get) Token: 0x0600007E RID: 126 RVA: 0x00003CAE File Offset: 0x00001EAE
		public bool CanMovePrevious
		{
			get
			{
				return this._searchResultList.Count<ReplaceResultInfo>() != 0 && this._searchResultList.IndexOf(this.CurrentSearchResult) != 0;
			}
		}

		// Token: 0x0600007F RID: 127 RVA: 0x00003CD8 File Offset: 0x00001ED8
		internal void MovePreviousResult()
		{
			if (this.CurrentSearchResult == null && this._searchResultList.Count<ReplaceResultInfo>() == 0)
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

		// Token: 0x06000080 RID: 128 RVA: 0x00003D52 File Offset: 0x00001F52
		internal void ClearSearchResult()
		{
			this._searchResultList.Clear();
			this.CurrentSearchResult = null;
			this.NotifyPropertyChanged("CurrentSearchResult");
		}

		// Token: 0x06000081 RID: 129 RVA: 0x00003D71 File Offset: 0x00001F71
		public void PublishSelectedEvent()
		{
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Publish(this.CurrentSearchResult.SearchInformation);
		}

		// Token: 0x06000082 RID: 130 RVA: 0x00003D8D File Offset: 0x00001F8D
		public void PublishReplaceEvent(ReplaceResultInfo info)
		{
			this.AddKeyWord(info.Keyword);
			this.AddReplaceAs(info.ReplaceAs);
			EventAggregatorManager.Global.GetEvent<ReplaceEvent>().Publish(info);
		}

		// Token: 0x06000083 RID: 131 RVA: 0x00003DB7 File Offset: 0x00001FB7
		public void RublishReplaceAllEvent(ReplaceResultInfo info)
		{
			this.AddKeyWord(info.Keyword);
			this.AddReplaceAs(info.ReplaceAs);
			EventAggregatorManager.Global.GetEvent<ReplaceAllEvent>().Publish(info);
		}

		// Token: 0x06000084 RID: 132 RVA: 0x00003DE4 File Offset: 0x00001FE4
		public void Search(string searchKey, bool isMatchCase, bool isEditableOnly, bool isSelectedOnly, string range)
		{
			this.ClearSearchResult();
			this.AddKeyWord(searchKey);
			this.ProgramKey = Application.Current.MainWindow.Tag as PackageKey;
			if (range.Equals("All Open Documents"))
			{
				this.ProgramKey = null;
			}
			SearchKeywordEventArgs e = new SearchKeywordEventArgs(this.ProgramKey, searchKey, isMatchCase, isEditableOnly, isSelectedOnly);
			this._lastSearchKeywordEventArgs = e;
			EventAggregatorManager.Global.GetEvent<ReplaceKeywordEvent>().Publish(e);
		}

		// Token: 0x06000085 RID: 133 RVA: 0x00003E78 File Offset: 0x00002078
		public void RemoveByProgramName(PackageKey key)
		{
			List<ReplaceResultInfo> list = this._searchResultList.Where<ReplaceResultInfo>((ReplaceResultInfo info) => info.SearchInformation.ProgramKey == key).ToList<ReplaceResultInfo>();
			foreach (ReplaceResultInfo replaceResultInfo in list)
			{
				this._searchResultList.Remove(replaceResultInfo);
			}
		}

		// Token: 0x04000026 RID: 38
		private ObservableCollection<string> _searchHistory = new ObservableCollection<string>();

		// Token: 0x04000027 RID: 39
		private ObservableCollection<ReplaceResultInfo> _searchResultList = new ObservableCollection<ReplaceResultInfo>();

		// Token: 0x04000029 RID: 41
		private string _replaceAs;

		// Token: 0x0400002A RID: 42
		private ObservableCollection<string> _replaceHistory = new ObservableCollection<string>();

		// Token: 0x0400002B RID: 43
		private ReplaceResultInfo _currentSearchResult;

		// Token: 0x0400002C RID: 44
		private SearchKeywordEventArgs _lastSearchKeywordEventArgs;

		// Token: 0x0400002D RID: 45
		private PackageKey ProgramKey;
	}
}
